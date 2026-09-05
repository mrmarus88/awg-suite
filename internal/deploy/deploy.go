// Package deploy contains the logic for the cross-platform SSH deploy tool
// (cmd/awg-deploy): parsing the target, building the remote commands that drive
// the non-interactive installer, and extracting the generated client config.
//
// The command/parsing logic is pure and unit-tested; the SSH transport lives in
// ssh.go and is exercised against a real server.
package deploy

import (
	"fmt"
	"sort"
	"strings"
)

const (
	beginMarker = "-----BEGIN_AWG_CONF-----"
	endMarker   = "-----END_AWG_CONF-----"
)

// Target identifies the server to deploy to.
type Target struct {
	User string
	Host string
	Port int
}

// Addr returns the host:port dial address.
func (t Target) Addr() string { return fmt.Sprintf("%s:%d", t.Host, t.Port) }

// ParseTarget parses "host", "user@host", or "user@host:port".
// User defaults to root, port to 22.
func ParseTarget(s string) (Target, error) {
	t := Target{User: "root", Port: 22}
	s = strings.TrimSpace(s)
	if s == "" {
		return t, fmt.Errorf("empty target")
	}
	if at := strings.LastIndex(s, "@"); at >= 0 {
		t.User = s[:at]
		s = s[at+1:]
		if t.User == "" {
			return t, fmt.Errorf("empty user in target")
		}
	}
	if colon := strings.LastIndex(s, ":"); colon >= 0 {
		host, portStr := s[:colon], s[colon+1:]
		p, err := parsePort(portStr)
		if err != nil {
			return t, err
		}
		t.Host, t.Port = host, p
	} else {
		t.Host = s
	}
	if t.Host == "" {
		return t, fmt.Errorf("empty host in target")
	}
	return t, nil
}

func parsePort(s string) (int, error) {
	p := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid port %q", s)
		}
		p = p*10 + int(r-'0')
	}
	if p < 1 || p > 65535 {
		return 0, fmt.Errorf("port out of range: %q", s)
	}
	return p, nil
}

// Sudo returns the command prefix for privilege escalation: empty for root,
// "sudo " otherwise.
func Sudo(user string) string {
	if user == "root" {
		return ""
	}
	return "sudo "
}

// InstallCommand builds the remote shell command that runs the installer
// (piped to bash via stdin) non-interactively with the given AWG_* env vars.
func InstallCommand(sudo string, env map[string]string) string {
	full := map[string]string{"AWG_PRINT_CONFIG": "1"}
	for k, v := range env {
		full[k] = v
	}
	return sudo + "env " + envArgs(full) + " bash -s -- --yes"
}

// AddClientCommand builds the remote command to create a single client. Optional
// env (e.g. AWG_ALLOWED_IPS / AWG_CLIENT_DNS / AWG_CLIENT_MTU for per-client
// overrides) is merged in; empty values are skipped.
func AddClientCommand(sudo, name string, env map[string]string) string {
	full := map[string]string{"AWG_PRINT_CONFIG": "1"}
	for k, v := range env {
		if v != "" {
			full[k] = v
		}
	}
	return sudo + "env " + envArgs(full) + " bash -s -- --add-client " + shellQuote(name)
}

// SetSecondaryEndpointCommand updates the optional fallback client endpoint on
// an existing server. It does not recreate the interface or any peer.
func SetSecondaryEndpointCommand(sudo, host string) string {
	return sudo + "env AWG_SERVER_IP_ALT=" + shellQuote(host) + " bash -s -- --set-secondary-endpoint " + shellQuote(host)
}

// RemoveClientCommand builds the remote command to delete a client.
func RemoveClientCommand(sudo, name string) string {
	return sudo + "bash -s -- --remove-client " + shellQuote(name)
}

// RenameClientCommand builds the remote command to rename a client.
func RenameClientCommand(sudo, oldName, newName string) string {
	return sudo + "bash -s -- --rename-client " + shellQuote(oldName) + " " + shellQuote(newName)
}

// ListClientsCommand builds the remote command to list clients.
func ListClientsCommand(sudo string) string {
	return sudo + "bash -s -- --list"
}

// UninstallCommand builds the remote command to remove everything. The
// AWG_CONFIRM=yes guard prevents an accidental wipe.
func UninstallCommand(sudo string) string {
	return sudo + "env AWG_CONFIRM=yes bash -s -- --uninstall"
}

// AlreadyInstalled reports whether installer output signals a configured server.
func AlreadyInstalled(output string) bool {
	return strings.Contains(output, "AWG_ALREADY_INSTALLED")
}

// CheckInstalledCommand prints AWG_INSTALLED if the server is already set up.
func CheckInstalledCommand(sudo string) string {
	return sudo + "test -f /etc/amnezia/amneziawg/params && echo AWG_INSTALLED || true"
}

// IsInstalled interprets the output of CheckInstalledCommand.
func IsInstalled(output string) bool {
	return strings.Contains(output, "AWG_INSTALLED")
}

// PanelPort is the TCP port the web panel listens on.
const PanelPort = 8443

// InstallPanelCommand builds the remote command to install the web panel
// non-interactively, passing the admin password via env.
func InstallPanelCommand(sudo, password string) string {
	return sudo + "env AWG_PANEL_PASSWORD=" + shellQuote(password) + " bash -s -- --install-panel"
}

// RemovePanelCommand builds the remote command to remove the web panel.
func RemovePanelCommand(sudo string) string {
	return sudo + "bash -s -- --remove-panel"
}

// PanelInstalledCommand prints AWG_PANEL_INSTALLED if the panel service exists.
func PanelInstalledCommand(sudo string) string {
	return sudo + "test -f /etc/systemd/system/awg-panel.service && echo AWG_PANEL_INSTALLED || true"
}

// IsPanelInstalled interprets the output of PanelInstalledCommand.
func IsPanelInstalled(output string) bool {
	return strings.Contains(output, "AWG_PANEL_INSTALLED")
}

// InstallBotCommand builds the remote command to install the Telegram bot
// non-interactively. Access needs both the allowlist and the password, so all
// three are passed via env.
func InstallBotCommand(sudo, token, admins, password string) string {
	return sudo + "env AWG_BOT_TOKEN=" + shellQuote(token) +
		" AWG_BOT_ADMINS=" + shellQuote(admins) +
		" AWG_BOT_PASSWORD=" + shellQuote(password) +
		" bash -s -- --install-bot"
}

// RemoveBotCommand builds the remote command to remove the Telegram bot.
func RemoveBotCommand(sudo string) string {
	return sudo + "bash -s -- --remove-bot"
}

// BotInstalledCommand prints AWG_BOT_INSTALLED if the bot service exists.
func BotInstalledCommand(sudo string) string {
	return sudo + "test -f /etc/systemd/system/awg-bot.service && echo AWG_BOT_INSTALLED || true"
}

// IsBotInstalled interprets the output of BotInstalledCommand.
func IsBotInstalled(output string) bool {
	return strings.Contains(output, "AWG_BOT_INSTALLED")
}

// Panel file locations on the server (mirrors the installer's readonly paths).
const (
	panelBin  = "/usr/local/bin/awg-panel"
	panelHash = "/etc/amnezia/amneziawg/panel.hash"
)

// PanelClientLimitsCommand lists every client's limits (quota/expiry/speed/
// disabled) as a JSON array, via the panel CLI. Requires the panel installed.
func PanelClientLimitsCommand(sudo string) string {
	return sudo + panelBin + " client-list"
}

// PanelSetLimitsCommand sets a client's quota (GB), expiry (days from now) and
// speed cap (Mbit/s) through the panel CLI. A zero value means unlimited/never.
// The panel daemon enforces them; this just records the desired state safely.
func PanelSetLimitsCommand(sudo, name string, quotaGB, expiresDays, speedMbit int) string {
	return sudo + panelBin + " client-set " + shellQuote(name) +
		" --quota-gb " + fmt.Sprint(quotaGB) +
		" --expires-days " + fmt.Sprint(expiresDays) +
		" --speed-mbit " + fmt.Sprint(speedMbit)
}

// PanelToggleClientCommand enables or disables a client through the panel CLI.
func PanelToggleClientCommand(sudo, name string, enable bool) string {
	sub := "client-disable"
	if enable {
		sub = "client-enable"
	}
	return sudo + panelBin + " " + sub + " " + shellQuote(name)
}

// ChangePanelPasswordCommand builds the remote command that rewrites the panel's
// bcrypt password hash and restarts the service. The new password is NOT part of
// the command: it is read from the command's stdin (pipe it via RunScript), so it
// never appears in argv, the process list, or logs.
func ChangePanelPasswordCommand(sudo string) string {
	script := "set -e; umask 077; IFS= read -r __pw; " +
		"printf '%s\\n' \"$__pw\" | " + panelBin + " hash > " + panelHash + "; " +
		"chmod 600 " + panelHash + "; systemctl restart awg-panel"
	return sudo + "bash -c " + shellQuote(script)
}

// ServerInfoCommand prints labelled KEY=VALUE lines describing the server: the
// WireGuard UDP port, AmneziaWG version, uptime (seconds) and peer count. Parse
// the output with ParseServerInfo.
func ServerInfoCommand(sudo, iface string) string {
	conf := "/etc/amnezia/amneziawg/" + iface + ".conf"
	script := "" +
		"echo PORT=$(grep -m1 -oE 'ListenPort[[:space:]]*=[[:space:]]*[0-9]+' " + shellQuote(conf) + " 2>/dev/null | grep -oE '[0-9]+'); " +
		"echo VER=$(awg --version 2>/dev/null | head -n1); " +
		"echo UP=$(cut -d. -f1 /proc/uptime 2>/dev/null); " +
		"echo PEERS=$(grep -c '^\\[Peer\\]' " + shellQuote(conf) + " 2>/dev/null)"
	return sudo + "bash -c " + shellQuote(script)
}

// ServerInfo holds the parsed fields from ServerInfoCommand output.
type ServerInfo struct {
	Port          string // WG UDP port ("" if unknown)
	Version       string // AmneziaWG version ("" if unknown)
	UptimeSeconds int64  // host uptime in seconds (0 if unknown)
	Peers         int    // number of configured peers
}

// ParseServerInfo reads the KEY=VALUE lines produced by ServerInfoCommand.
func ParseServerInfo(output string) ServerInfo {
	var info ServerInfo
	for _, line := range strings.Split(output, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch k {
		case "PORT":
			info.Port = v
		case "VER":
			info.Version = v
		case "UP":
			info.UptimeSeconds = atoi64(v)
		case "PEERS":
			info.Peers = int(atoi64(v))
		}
	}
	return info
}

func atoi64(s string) int64 {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

// MonitorDumpCommand returns the command that dumps live interface state.
func MonitorDumpCommand(sudo, iface string) string {
	return sudo + "awg show " + shellQuote(iface) + " dump"
}

// ReadConfCommand returns the command that prints the server config (for client
// name resolution in monitor mode).
func ReadConfCommand(sudo, iface string) string {
	return sudo + "cat /etc/amnezia/amneziawg/" + iface + ".conf"
}

// ExtractConfig pulls the fenced client config out of installer output.
func ExtractConfig(output string) (string, error) {
	start := strings.Index(output, beginMarker)
	end := strings.Index(output, endMarker)
	if start < 0 || end < 0 || end < start {
		return "", fmt.Errorf("client config markers not found in output")
	}
	conf := output[start+len(beginMarker) : end]
	return strings.TrimSpace(conf) + "\n", nil
}

// envArgs renders a deterministic "K=quoted ..." string for `env`.
func envArgs(env map[string]string) string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+shellQuote(env[k]))
	}
	return strings.Join(parts, " ")
}

// shellQuote single-quotes a value for safe use in a remote shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
