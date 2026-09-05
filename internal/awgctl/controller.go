package awgctl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hennessyxo/awg-suite/internal/awg"
	"github.com/hennessyxo/awg-suite/internal/lifecycle"
)

// Client is a generated VPN client and its full configuration text.
type Client struct {
	Name   string
	Config string
}

// AddOptions carries optional lifecycle limits and per-client config overrides
// for a new client. The override fields are empty by default (server defaults).
type AddOptions struct {
	ExpiresIn  time.Duration // 0 = never expires
	QuotaBytes uint64        // 0 = unlimited
	SpeedMbit  int           // bandwidth cap in Mbit/s (0 = unlimited)

	// Advanced per-client config overrides ("" = server default).
	AllowedIPs string // client routing (split tunnel)
	DNS        string // custom DNS
	MTU        string // custom MTU
}

// UpdateOptions carries the new lifecycle limits for an existing client.
type UpdateOptions struct {
	ExpiresIn  time.Duration // 0 = no expiry
	QuotaBytes uint64        // 0 = unlimited
	SpeedMbit  int           // 0 = unlimited
}

// Controller is the panel's view of the running AmneziaWG server. It is an
// interface so HTTP handlers can be tested against a fake.
type Controller interface {
	Snapshot() (awg.Snapshot, error)
	AddClient(name string, opts AddOptions) (Client, error)
	UpdateClient(name string, opts UpdateOptions) error
	RenameClient(oldName, newName string) error
	RevokeClient(name string) error
	DisableClient(name string) error
	EnableClient(name string) error
	ClientConfig(name string) (string, error)
	ServerClients() ([]ServerClient, error)
}

// FileController is the production Controller: it shells out to `awg`, edits the
// server config on disk, and records lifecycle metadata in the store. All the
// config-rewriting logic lives in the unit-tested pure functions above.
type FileController struct {
	Iface     string           // e.g. "awg0"
	ConfPath  string           // /etc/amnezia/amneziawg/awg0.conf
	ParamPath string           // /etc/amnezia/amneziawg/params
	ClientDir string           // where panel-generated client .conf files are stored
	Store     *lifecycle.Store // lifecycle metadata (quotas, expiry)
}

func (c FileController) Snapshot() (awg.Snapshot, error) {
	out, err := exec.Command("awg", "show", c.Iface, "dump").Output()
	if err != nil {
		return awg.Snapshot{}, fmt.Errorf("awg show: %w", err)
	}
	snap, err := awg.ParseDump(c.Iface, string(out), time.Now())
	if err != nil {
		return snap, err
	}
	if data, e := os.ReadFile(c.ConfPath); e == nil {
		awg.ApplyNames(snap.Peers, awg.ParseNames(string(data)))
	}
	return snap, nil
}

func (c FileController) AddClient(name string, opts AddOptions) (Client, error) {
	confBytes, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return Client{}, err
	}
	conf := string(confBytes)
	if HasPeer(conf, name) {
		return Client{}, fmt.Errorf("client %q already exists", name)
	}

	paramBytes, err := os.ReadFile(c.ParamPath)
	if err != nil {
		return Client{}, err
	}
	params := ParseParams(string(paramBytes))

	priv, err := c.genKey()
	if err != nil {
		return Client{}, err
	}
	pub, err := c.pubKey(priv)
	if err != nil {
		return Client{}, err
	}
	psk, err := c.genPSK()
	if err != nil {
		return Client{}, err
	}
	octet, err := FreeOctet(conf, c.reservedOctets())
	if err != nil {
		return Client{}, err
	}

	ov := ClientOverrides{AllowedIPs: opts.AllowedIPs, DNS: opts.DNS, MTU: opts.MTU}
	clientCfg := RenderClientConfig(params, ov, priv, psk, octet)
	block := PeerBlock(name, pub, psk, octet)

	if err := os.WriteFile(c.ConfPath, []byte(AppendBlock(conf, block)), 0o600); err != nil {
		return Client{}, err
	}
	if err := os.MkdirAll(c.ClientDir, 0o700); err != nil {
		return Client{}, err
	}
	if err := os.WriteFile(c.clientFile(name), []byte(clientCfg), 0o600); err != nil {
		return Client{}, err
	}
	if params.ServerPubIPAlt != "" {
		altCfg := WithEndpoint(clientCfg, params.ServerPubIPAlt, params.ServerPort)
		if err := os.WriteFile(c.secondaryClientFile(name), []byte(altCfg), 0o600); err != nil {
			return Client{}, err
		}
	}
	if err := c.syncConf(); err != nil {
		return Client{}, err
	}

	c.record(name, pub, octet, block, opts)
	return Client{Name: name, Config: clientCfg}, nil
}

func (c FileController) RevokeClient(name string) error {
	confBytes, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return err
	}
	newConf, _ := RemovePeerBlock(string(confBytes), name)
	if err := os.WriteFile(c.ConfPath, []byte(newConf), 0o600); err != nil {
		return err
	}
	_ = os.Remove(c.clientFile(name))
	_ = os.Remove(c.secondaryClientFile(name))
	if c.Store != nil {
		_ = c.Store.Delete(name)
	}
	return c.syncConf()
}

// DisableClient cuts a client off the live interface (and config) but keeps its
// record and stored peer block so it can be re-enabled.
func (c FileController) DisableClient(name string) error {
	confBytes, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return err
	}
	newConf, _ := RemovePeerBlock(string(confBytes), name)
	if err := os.WriteFile(c.ConfPath, []byte(newConf), 0o600); err != nil {
		return err
	}
	if c.Store != nil {
		_ = c.Store.Mutate(name, func(rec *lifecycle.Record) { rec.Disabled = true })
	}
	return c.syncConf()
}

// EnableClient re-adds a previously disabled client from its stored peer block.
func (c FileController) EnableClient(name string) error {
	if c.Store == nil {
		return fmt.Errorf("no lifecycle store configured")
	}
	rec, ok := c.Store.Get(name)
	if !ok {
		return fmt.Errorf("client %q not found", name)
	}
	confBytes, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return err
	}
	conf := string(confBytes)
	if !HasPeer(conf, name) {
		if err := os.WriteFile(c.ConfPath, []byte(AppendBlock(conf, rec.PeerBlock)), 0o600); err != nil {
			return err
		}
	}
	// Clear whatever caused the auto-disable so the enforcer doesn't immediately
	// re-disable: drop a past expiry, and reset usage if it was over quota. Done
	// inside Mutate so it operates on the record's latest stored state.
	now := time.Now()
	_ = c.Store.Mutate(name, func(rec *lifecycle.Record) {
		if rec.ExpiresAt != nil && now.After(*rec.ExpiresAt) {
			rec.ExpiresAt = nil
		}
		if rec.QuotaBytes > 0 && rec.UsedBytes >= rec.QuotaBytes {
			rec.UsedBytes = 0
			rec.LastRx, rec.LastTx = 0, 0
		}
		rec.Disabled = false
	})
	return c.syncConf()
}

// UpdateClient changes an existing client's lifecycle limits (speed/quota/expiry).
func (c FileController) UpdateClient(name string, opts UpdateOptions) error {
	if c.Store == nil {
		return fmt.Errorf("no lifecycle store configured")
	}
	if _, ok := c.Store.Get(name); !ok {
		return fmt.Errorf("client %q not found", name)
	}
	now := time.Now()
	return c.Store.Mutate(name, func(rec *lifecycle.Record) {
		rec.QuotaBytes = opts.QuotaBytes
		rec.SpeedMbit = opts.SpeedMbit
		if opts.ExpiresIn > 0 {
			exp := now.Add(opts.ExpiresIn)
			rec.ExpiresAt = &exp
		} else {
			rec.ExpiresAt = nil
		}
	})
}

// RenameClient renames a client across the server config, its config file, and
// the lifecycle store.
func (c FileController) RenameClient(oldName, newName string) error {
	confBytes, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return err
	}
	conf := string(confBytes)
	if HasPeer(conf, newName) {
		return fmt.Errorf("client %q already exists", newName)
	}
	if c.Store != nil {
		if _, exists := c.Store.Get(newName); exists {
			return fmt.Errorf("client %q already exists", newName)
		}
	}

	if newConf := RenamePeer(conf, oldName, newName); newConf != conf {
		if err := os.WriteFile(c.ConfPath, []byte(newConf), 0o600); err != nil {
			return err
		}
	}
	_ = os.Rename(c.clientFile(oldName), c.clientFile(newName))
	_ = os.Rename(c.secondaryClientFile(oldName), c.secondaryClientFile(newName))

	if c.Store != nil {
		if rec, ok := c.Store.Get(oldName); ok {
			rec.Name = newName
			rec.PeerBlock = RenamePeer(rec.PeerBlock, oldName, newName)
			_ = c.Store.Delete(oldName)
			_ = c.Store.Put(rec)
		}
	}
	return c.syncConf()
}

// ServerClients parses the peers currently in the server config.
func (c FileController) ServerClients() ([]ServerClient, error) {
	data, err := os.ReadFile(c.ConfPath)
	if err != nil {
		return nil, err
	}
	return ParseServerClients(string(data)), nil
}

func (c FileController) ClientConfig(name string) (string, error) {
	data, err := os.ReadFile(c.clientFile(name))
	if err != nil {
		return "", fmt.Errorf("config for %q unavailable (only panel-created clients are stored): %w", name, err)
	}
	return string(data), nil
}

// ClientConfigForEndpoint exports the configured fallback profile without
// creating a new peer. endpoint must be "primary" or "secondary".
func (c FileController) ClientConfigForEndpoint(name, endpoint string) (string, error) {
	cfg, err := c.ClientConfig(name)
	if err != nil || endpoint == "" || endpoint == "primary" {
		return cfg, err
	}
	if endpoint != "secondary" {
		return "", fmt.Errorf("unknown endpoint profile %q", endpoint)
	}
	paramsBytes, err := os.ReadFile(c.ParamPath)
	if err != nil {
		return "", err
	}
	params := ParseParams(string(paramsBytes))
	if params.ServerPubIPAlt == "" {
		return "", fmt.Errorf("secondary endpoint is not configured")
	}
	return WithEndpoint(cfg, params.ServerPubIPAlt, params.ServerPort), nil
}

// HasSecondaryEndpoint reports whether this server has an optional fallback
// endpoint. Read errors deliberately look like false to keep old installs and
// callers that do not need the feature working as before.
func (c FileController) HasSecondaryEndpoint() bool {
	data, err := os.ReadFile(c.ParamPath)
	return err == nil && ParseParams(string(data)).ServerPubIPAlt != ""
}

// --- helpers ---------------------------------------------------------------

func (c FileController) reservedOctets() map[int]bool {
	if c.Store == nil {
		return nil
	}
	return c.Store.UsedOctets()
}

func (c FileController) record(name, pub string, octet int, block string, opts AddOptions) {
	if c.Store == nil {
		return
	}
	rec := lifecycle.Record{
		Name:       name,
		PubKey:     pub,
		Octet:      octet,
		PeerBlock:  block,
		CreatedAt:  time.Now(),
		QuotaBytes: opts.QuotaBytes,
		SpeedMbit:  opts.SpeedMbit,
	}
	if opts.ExpiresIn > 0 {
		exp := time.Now().Add(opts.ExpiresIn)
		rec.ExpiresAt = &exp
	}
	_ = c.Store.Put(rec)
}

func (c FileController) clientFile(name string) string {
	return filepath.Join(c.ClientDir, c.Iface+"-client-"+name+".conf")
}

func (c FileController) secondaryClientFile(name string) string {
	return filepath.Join(c.ClientDir, c.Iface+"-client-"+name+"-secondary.conf")
}

func (c FileController) genKey() (string, error) { return runOut("awg", "genkey") }
func (c FileController) genPSK() (string, error) { return runOut("awg", "genpsk") }

func (c FileController) pubKey(priv string) (string, error) {
	cmd := exec.Command("awg", "pubkey")
	cmd.Stdin = strings.NewReader(priv + "\n")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("awg pubkey: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// syncConf applies config changes to the live interface without dropping peers.
func (c FileController) syncConf() error {
	stripped, err := exec.Command("awg-quick", "strip", c.Iface).Output()
	if err != nil {
		return fmt.Errorf("awg-quick strip: %w", err)
	}
	tmp, err := os.CreateTemp("", "awg-sync-*.conf")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(stripped); err != nil {
		return err
	}
	tmp.Close()
	if err := exec.Command("awg", "syncconf", c.Iface, tmp.Name()).Run(); err != nil {
		return fmt.Errorf("awg syncconf: %w", err)
	}
	return nil
}

func runOut(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}
