package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	amneziawg "github.com/hennessyxo/awg-suite"
	"github.com/hennessyxo/awg-suite/internal/awgctl"
	"github.com/hennessyxo/awg-suite/internal/deploy"
	"github.com/skip2/go-qrcode"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// serverConf is the AmneziaWG server config path; the interface name is a fixed
// constant in the installer (awg0).
const serverConf = "/etc/amnezia/amneziawg/awg0.conf"

// clientConfDir is where the installer mirrors each client's .conf (PANEL_CLIENT_DIR).
const clientConfDir = "/etc/amnezia/amneziawg/clients"

// App is the Wails-bound backend. Its exported methods are callable from the
// frontend as window.go.main.App.*.
type App struct {
	ctx context.Context

	mu       sync.Mutex
	client   *deploy.Client
	target   deploy.Target
	keepStop chan struct{} // closes to stop the keepalive loop
	lang     string        // "ru" | "en" — for server-formatted strings
}

// NewApp constructs the backend in its disconnected state.
func NewApp() *App { return &App{lang: "ru"} }

// SetLang sets the language used for backend-formatted strings (uptime/handshake).
func (a *App) SetLang(lang string) {
	if lang == "en" {
		a.lang = "en"
	} else {
		a.lang = "ru"
	}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }
func (a *App) shutdown(_ context.Context)  { a.closeClient() }

func (a *App) closeClient() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.keepStop != nil {
		close(a.keepStop)
		a.keepStop = nil
	}
	if a.client != nil {
		_ = a.client.Close()
		a.client = nil
	}
}

// keepAliveLoop pings the server periodically so an idle SSH session isn't
// dropped by the server or a NAT timeout. It exits when stop is closed.
func (a *App) keepAliveLoop(stop chan struct{}) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			a.mu.Lock()
			cl := a.client
			a.mu.Unlock()
			if cl == nil {
				return
			}
			_ = cl.KeepAlive()
		}
	}
}

// --- request / response types (JSON-bound to the frontend) -----------------

// ConnectRequest carries the SSH connection parameters from the UI.
type ConnectRequest struct {
	Host         string `json:"host"`
	User         string `json:"user"`
	Label        string `json:"label"` // optional friendly name
	Password     string `json:"password"`
	IdentityPath string `json:"identityPath"`
	AuthMode     string `json:"authMode"` // "password" | "key"
	Remember     bool   `json:"remember"`
}

// InstallRequest carries the install options from the UI.
type InstallRequest struct {
	Port              string `json:"port"`              // optional UDP port (blank = auto/free)
	Client            string `json:"client"`            // first client name
	Preset            string `json:"preset"`            // obfuscation profile: mobile|desktop|plain (blank = mobile)
	Dns1              string `json:"dns1"`              // optional custom DNS (blank = default)
	Dns2              string `json:"dns2"`              // optional secondary DNS
	SecondaryEndpoint string `json:"secondaryEndpoint"` // optional second public IP/host
}

// StatusResult reports whether the connected server already has AmneziaWG.
type StatusResult struct {
	Installed bool `json:"installed"`
}

// PanelResult reports the web panel state and the URL to reach it.
type PanelResult struct {
	Installed bool   `json:"installed"`
	URL       string `json:"url"`
}

// ClientResult is a freshly created client's config plus a scannable QR image
// rendered as a data URI for direct use in an <img src>.
type ClientResult struct {
	Name    string `json:"name"`
	Conf    string `json:"conf"`
	QR      string `json:"qr"`
	ConfAlt string `json:"confAlt,omitempty"`
	QRAlt   string `json:"qrAlt,omitempty"`
}

// --- bound methods ---------------------------------------------------------

// Connect opens an SSH session to the server and stores it for later calls.
func (a *App) Connect(req ConnectRequest) error {
	host := strings.TrimSpace(req.Host)
	if host == "" {
		return fmt.Errorf("укажите адрес сервера")
	}
	user := strings.TrimSpace(req.User)
	if user == "" {
		user = "root"
	}
	t, err := deploy.ParseTarget(user + "@" + host)
	if err != nil {
		return err
	}

	cl, err := dial(t, strings.TrimSpace(req.IdentityPath), req.Password)
	if err != nil {
		return err
	}

	a.closeClient()
	a.mu.Lock()
	a.client = cl
	a.target = t
	stop := make(chan struct{})
	a.keepStop = stop
	a.mu.Unlock()
	go a.keepAliveLoop(stop)

	a.persistPrefs(req, host, user)
	return nil
}

// persistPrefs saves the server as a profile (non-secret fields to disk) and,
// when the user asked to remember, the password to the OS secret store.
func (a *App) persistPrefs(req ConnectRequest, host, user string) {
	upsertProfile(ProfileEntry{
		Host:         host,
		User:         user,
		Label:        strings.TrimSpace(req.Label),
		AuthMode:     req.AuthMode,
		IdentityPath: strings.TrimSpace(req.IdentityPath),
		Remember:     req.Remember,
	})
	if req.Remember && req.AuthMode != "key" && req.Password != "" {
		_ = rememberPassword(user, host, req.Password)
	} else {
		forgetPassword(user, host)
	}
}

// LoadPrefs returns the last-used server for prefilling the form (password from
// the secret store only if "remember" was set).
func (a *App) LoadPrefs() (Prefs, error) {
	d := loadProfilesDisk()
	for _, e := range d.Profiles {
		if profileKey(e.User, e.Host) == d.Last {
			return e.asPrefs(), nil
		}
	}
	return Prefs{}, nil
}

// ListProfiles returns every saved server (with password pulled from the secret
// store where remembered) so the UI can offer one-click reconnect.
func (a *App) ListProfiles() ([]Prefs, error) {
	d := loadProfilesDisk()
	out := make([]Prefs, 0, len(d.Profiles))
	for _, e := range d.Profiles {
		out = append(out, e.asPrefs())
	}
	return out, nil
}

// DeleteProfile removes a saved server and forgets its stored password.
func (a *App) DeleteProfile(host, user string) error {
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	if user == "" {
		user = "root"
	}
	removeProfile(user, host)
	return nil
}

// Disconnect closes the SSH session.
func (a *App) Disconnect() error {
	a.closeClient()
	return nil
}

// ServerStatus reports whether the server already has AmneziaWG installed.
func (a *App) ServerStatus() (StatusResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return StatusResult{}, err
	}
	out, err := cl.Run(deploy.CheckInstalledCommand(deploy.Sudo(t.User)))
	if err != nil {
		return StatusResult{}, fmt.Errorf("проверка сервера не удалась: %w", err)
	}
	return StatusResult{Installed: deploy.IsInstalled(out)}, nil
}

// Install runs the installer on the server, streaming progress to the UI via
// the "install:log" event, and returns the first client's config + QR.
func (a *App) Install(req InstallRequest) (ClientResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return ClientResult{}, err
	}
	client := strings.TrimSpace(req.Client)
	if client == "" {
		client = "phone"
	}

	env := map[string]string{"AWG_PRESET": normalizePreset(req.Preset), "AWG_CLIENT": client}
	if p := strings.TrimSpace(req.Port); p != "" {
		env["AWG_PORT"] = p
	}
	if d := strings.TrimSpace(req.Dns1); d != "" {
		env["AWG_DNS1"] = d
	}
	if d := strings.TrimSpace(req.Dns2); d != "" {
		env["AWG_DNS2"] = d
	}
	if endpoint := strings.TrimSpace(req.SecondaryEndpoint); endpoint != "" {
		if !validEndpointHost(endpoint) {
			return ClientResult{}, fmt.Errorf("резервный Endpoint должен быть IPv4-адресом или именем хоста")
		}
		env["AWG_SERVER_IP_ALT"] = endpoint
	}

	out, err := cl.RunScript(deploy.InstallCommand(deploy.Sudo(t.User), env), amneziawg.InstallerScript, a.logWriter("install:log"))
	if err != nil {
		return ClientResult{}, fmt.Errorf("установка не удалась: %w", err)
	}
	if deploy.AlreadyInstalled(out) {
		return ClientResult{}, fmt.Errorf("сервер уже настроен — используйте управление клиентами")
	}
	return buildClientResult(client, out)
}

// AddClient creates a new client on the server and returns its config + QR.
// endpointProfile may be "primary", "secondary", or "both" (the default).
func (a *App) AddClient(name, allowedIPs, dns, mtu, endpointProfile string) (ClientResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return ClientResult{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ClientResult{}, fmt.Errorf("укажите имя клиента")
	}
	// Optional per-client overrides ("" = server default). Sanitized so nothing
	// can inject extra lines into the generated client config.
	env := map[string]string{}
	if v := safeConfList(allowedIPs); v != "" {
		env["AWG_ALLOWED_IPS"] = v
	}
	if v := safeConfList(dns); v != "" {
		env["AWG_CLIENT_DNS"] = v
	}
	if v := safeMTU(mtu); v != "" {
		env["AWG_CLIENT_MTU"] = v
	}
	endpointProfile = strings.TrimSpace(endpointProfile)
	if endpointProfile != "" && endpointProfile != "both" {
		if endpointProfile != "primary" && endpointProfile != "secondary" {
			return ClientResult{}, fmt.Errorf("неизвестный профиль Endpoint")
		}
		env["AWG_CLIENT_PROFILE"] = endpointProfile
	}
	out, err := cl.RunScript(deploy.AddClientCommand(deploy.Sudo(t.User), name, env), amneziawg.InstallerScript, a.logWriter("client:log"))
	if err != nil {
		return ClientResult{}, fmt.Errorf("создание клиента не удалось: %w", err)
	}
	return buildClientResult(name, out)
}

// safeConfList accepts an AllowedIPs/DNS value only if it is just the safe charset
// (IP digits/hex, dots, colons, slashes, commas, spaces) — no newline injection.
func safeConfList(s string) string {
	s = strings.TrimSpace(s)
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F',
			r == '.', r == ':', r == '/', r == ',', r == ' ':
		default:
			return ""
		}
	}
	return s
}

// safeMTU returns the MTU only if it is a plausible value, else "" (default).
func safeMTU(s string) string {
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil || n < 576 || n > 9000 {
		return ""
	}
	return s
}

// SaveConfig opens a native Save dialog and writes the client config to the
// chosen path. The blob/<a download> trick doesn't work inside the webview, so
// saving goes through the Wails runtime instead. Returns "" path on cancel.
func (a *App) SaveConfig(name, conf string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("приложение не готово")
	}
	// Match the panel's naming (awg0-client-<name>.conf) and constrain the dialog
	// to .conf so the OS keeps the extension.
	defaultName := fmt.Sprintf("%s-client-%s.conf", awgIface, name)
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "Сохранить конфиг клиента",
		Filters: []wruntime.FileFilter{
			{DisplayName: "AmneziaWG config (*.conf)", Pattern: "*.conf"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("диалог сохранения не удался: %w", err)
	}
	if path == "" {
		return "", nil // user cancelled
	}
	// Guarantee a .conf extension even if the OS dialog didn't append it, so the
	// file imports cleanly into the AmneziaWG app.
	if !strings.HasSuffix(strings.ToLower(path), ".conf") {
		path += ".conf"
	}
	if err := os.WriteFile(path, []byte(conf), 0o600); err != nil {
		return "", fmt.Errorf("не удалось сохранить файл: %w", err)
	}
	return path, nil
}

// ClientConfig reads an existing client's mirrored .conf from the server and
// returns it with a QR — so the config/QR is available any time, not just at
// creation. The installer mirrors each client into clientConfDir.
func (a *App) ClientConfig(name string) (ClientResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return ClientResult{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ClientResult{}, fmt.Errorf("укажите имя клиента")
	}
	path := fmt.Sprintf("%s/awg0-client-%s.conf", clientConfDir, name)
	out, err := cl.Run(deploy.Sudo(t.User) + "cat " + shellQuote(path))
	if err != nil || strings.TrimSpace(out) == "" {
		return ClientResult{}, fmt.Errorf("конфиг клиента не найден на сервере (возможно, он создан вне установщика)")
	}
	primary := strings.TrimSpace(out) + "\n"
	paramsOut, _ := cl.Run(deploy.Sudo(t.User) + "cat /etc/amnezia/amneziawg/params 2>/dev/null || true")
	params := awgctl.ParseParams(paramsOut)
	alt := ""
	if params.ServerPubIPAlt != "" {
		// Generate it on demand instead of relying on a second file. This lets a
		// server that already had clients gain fallback exports immediately after
		// SetSecondaryEndpoint, with no peer/key changes or reinstallation.
		alt = awgctl.WithEndpoint(primary, params.ServerPubIPAlt, params.ServerPort)
	}
	return clientResultFromConfs(name, primary, alt), nil
}

// SetSecondaryEndpoint changes only the optional fallback address in params.
// Existing peers and their primary profiles keep working; new/exported profiles
// can then use the same keys through the new address.
func (a *App) SetSecondaryEndpoint(endpoint string) error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" && !validEndpointHost(endpoint) {
		return fmt.Errorf("резервный Endpoint должен быть IPv4-адресом или именем хоста")
	}
	out, err := cl.RunScript(deploy.SetSecondaryEndpointCommand(deploy.Sudo(t.User), endpoint), amneziawg.InstallerScript, a.logWriter("endpoint:log"))
	if err != nil {
		return fmt.Errorf("не удалось сохранить резервный Endpoint: %w\n%s", err, out)
	}
	return nil
}

func validEndpointHost(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// RenameClient renames a client on the server.
func (a *App) RenameClient(oldName, newName string) error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("укажите имя клиента")
	}
	out, err := cl.RunScript(deploy.RenameClientCommand(deploy.Sudo(t.User), oldName, newName), amneziawg.InstallerScript, a.logWriter("client:log"))
	if err != nil {
		return fmt.Errorf("переименование не удалось: %w\n%s", err, out)
	}
	return nil
}

// RemoveClient deletes a client from the server.
func (a *App) RemoveClient(name string) error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("укажите имя клиента")
	}
	out, err := cl.RunScript(deploy.RemoveClientCommand(deploy.Sudo(t.User), name), amneziawg.InstallerScript, a.logWriter("client:log"))
	if err != nil {
		return fmt.Errorf("удаление не удалось: %w\n%s", err, out)
	}
	return nil
}

// ListClients returns the names of all clients configured on the server.
func (a *App) ListClients() ([]string, error) {
	cl, t, err := a.conn()
	if err != nil {
		return nil, err
	}
	out, err := cl.Run(deploy.Sudo(t.User) + "cat " + serverConf + " 2>/dev/null || true")
	if err != nil {
		return nil, fmt.Errorf("чтение конфигурации не удалось: %w", err)
	}
	clients := awgctl.ParseServerClients(out)
	names := make([]string, 0, len(clients))
	for _, c := range clients {
		names = append(names, c.Name)
	}
	sort.Strings(names)
	return names, nil
}

// panelURL builds the https URL the web panel is reachable at (the host the
// user connected to, on the panel port).
func (a *App) panelURL() string {
	return fmt.Sprintf("https://%s:%d", a.target.Host, deploy.PanelPort)
}

// PanelStatus reports whether the web panel is installed and its URL.
func (a *App) PanelStatus() (PanelResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return PanelResult{}, err
	}
	out, err := cl.Run(deploy.PanelInstalledCommand(deploy.Sudo(t.User)))
	if err != nil {
		return PanelResult{}, fmt.Errorf("проверка панели не удалась: %w", err)
	}
	return PanelResult{Installed: deploy.IsPanelInstalled(out), URL: a.panelURL()}, nil
}

// InstallPanel installs the web panel non-interactively with the given admin
// password (min 8 chars) and returns its URL. The panel carries the advanced
// per-client limits (speed/quota/expiry) and the enforcer daemon.
func (a *App) InstallPanel(password string) (PanelResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return PanelResult{}, err
	}
	if !validPanelPassword(password) {
		return PanelResult{}, fmt.Errorf("слабый пароль: минимум 6 символов, строчные и заглавные буквы, цифра и спецсимвол (например Admin2@)")
	}
	out, err := cl.RunScript(deploy.InstallPanelCommand(deploy.Sudo(t.User), password), amneziawg.InstallerScript, a.logWriter("panel:log"))
	if err != nil {
		return PanelResult{}, fmt.Errorf("установка панели не удалась: %w\n%s", err, out)
	}
	return PanelResult{Installed: true, URL: a.panelURL()}, nil
}

// ClientLimit mirrors the JSON from `awg-panel client-list`: the per-client
// limits the panel daemon enforces. The desktop app reads these to prefill the
// limit controls and writes them back through SetClientLimits.
type ClientLimit struct {
	Name        string `json:"name"`
	Disabled    bool   `json:"disabled"`
	QuotaGB     int    `json:"quotaGB"`
	UsedBytes   uint64 `json:"usedBytes"`
	SpeedMbit   int    `json:"speedMbit"`
	ExpiresDays int    `json:"expiresDays"`
}

// ClientLimits lists every client's limits via the panel CLI over SSH. It only
// works when the web panel is installed (it owns the enforcing daemon); without
// it the call returns an empty list so the UI can hide the controls.
func (a *App) ClientLimits() ([]ClientLimit, error) {
	cl, t, err := a.conn()
	if err != nil {
		return nil, err
	}
	out, err := cl.Run(deploy.PanelClientLimitsCommand(deploy.Sudo(t.User)))
	if err != nil {
		return nil, fmt.Errorf("не удалось получить лимиты: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return []ClientLimit{}, nil
	}
	var limits []ClientLimit
	if err := json.Unmarshal([]byte(out), &limits); err != nil {
		return nil, fmt.Errorf("не удалось разобрать лимиты (установлена ли веб-панель?): %w", err)
	}
	return limits, nil
}

// SetClientLimits sets a client's quota (GB), expiry (days from now) and speed
// cap (Mbit/s) through the panel CLI. Zero means unlimited/never. Enforcement is
// done by the panel's always-on daemon, so the panel must be installed.
func (a *App) SetClientLimits(name string, quotaGB, expiresDays, speedMbit int) error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("укажите имя клиента")
	}
	cmd := deploy.PanelSetLimitsCommand(deploy.Sudo(t.User), name,
		clampLimit(quotaGB), clampLimit(expiresDays), clampLimit(speedMbit))
	if out, err := cl.Run(cmd); err != nil {
		return fmt.Errorf("не удалось задать лимиты: %w\n%s", err, strings.TrimSpace(out))
	}
	return nil
}

// SetClientEnabled enables or disables a client through the panel CLI.
func (a *App) SetClientEnabled(name string, enabled bool) error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("укажите имя клиента")
	}
	cmd := deploy.PanelToggleClientCommand(deploy.Sudo(t.User), name, enabled)
	if out, err := cl.Run(cmd); err != nil {
		return fmt.Errorf("не удалось изменить статус клиента: %w\n%s", err, strings.TrimSpace(out))
	}
	return nil
}

// clampLimit keeps a UI-provided limit in a sane non-negative range.
func clampLimit(n int) int {
	if n < 0 {
		return 0
	}
	if n > 1000000 {
		return 1000000
	}
	return n
}

// RemovePanel removes the web panel from the server.
func (a *App) RemovePanel() error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	out, err := cl.RunScript(deploy.RemovePanelCommand(deploy.Sudo(t.User)), amneziawg.InstallerScript, a.logWriter("panel:log"))
	if err != nil {
		return fmt.Errorf("удаление панели не удалось: %w\n%s", err, out)
	}
	return nil
}

// BotResult reports whether the Telegram bot is installed.
type BotResult struct {
	Installed bool `json:"installed"`
}

// BotStatus reports whether the Telegram bot is installed on the server.
func (a *App) BotStatus() (BotResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return BotResult{}, err
	}
	out, err := cl.Run(deploy.BotInstalledCommand(deploy.Sudo(t.User)))
	if err != nil {
		return BotResult{}, fmt.Errorf("проверка бота не удалась: %w", err)
	}
	return BotResult{Installed: deploy.IsBotInstalled(out)}, nil
}

// InstallBot installs the Telegram bot non-interactively. Access requires BOTH
// the admin allowlist and the access password.
func (a *App) InstallBot(token, admins, password string) error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	token = strings.TrimSpace(token)
	admins = strings.TrimSpace(admins)
	if token == "" {
		return fmt.Errorf("укажите токен бота (получите его у @BotFather)")
	}
	if admins == "" {
		return fmt.Errorf("укажите хотя бы один Telegram ID (узнать у @userinfobot)")
	}
	if !validPanelPassword(password) {
		return fmt.Errorf("слабый пароль: минимум 6 символов, строчные и заглавные буквы, цифра и спецсимвол (например Admin2@)")
	}
	out, err := cl.RunScript(deploy.InstallBotCommand(deploy.Sudo(t.User), token, admins, password), amneziawg.InstallerScript, a.logWriter("bot:log"))
	if err != nil {
		return fmt.Errorf("установка бота не удалась: %w\n%s", err, out)
	}
	return nil
}

// RemoveBot removes the Telegram bot from the server.
func (a *App) RemoveBot() error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	out, err := cl.RunScript(deploy.RemoveBotCommand(deploy.Sudo(t.User)), amneziawg.InstallerScript, a.logWriter("bot:log"))
	if err != nil {
		return fmt.Errorf("удаление бота не удалось: %w\n%s", err, out)
	}
	return nil
}

// OpenPanel opens the web panel URL in the user's default browser.
func (a *App) OpenPanel() error {
	if a.ctx == nil {
		return fmt.Errorf("приложение не готово")
	}
	wruntime.BrowserOpenURL(a.ctx, a.panelURL())
	return nil
}

// Uninstall removes AmneziaWG and everything it set up from the server.
func (a *App) Uninstall() error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	out, err := cl.RunScript(deploy.UninstallCommand(deploy.Sudo(t.User)), amneziawg.InstallerScript, a.logWriter("install:log"))
	if err != nil {
		return fmt.Errorf("удаление не удалось: %w\n%s", err, out)
	}
	return nil
}

// RunCommand executes a single shell command on the connected server and returns
// its combined stdout+stderr. Each call runs in a fresh SSH session, so there is
// no persistent shell state or working directory between commands — this powers
// the in-app "server terminal". A non-zero exit status is normal for a terminal
// (e.g. grep with no match), so its output is returned rather than treated as an
// error; only a session/transport failure with no output surfaces as an error.
func (a *App) RunCommand(cmd string) (string, error) {
	cl, _, err := a.conn()
	if err != nil {
		return "", err
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", nil
	}
	out, runErr := cl.Run(cmd)
	if runErr != nil && strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("команда не выполнена: %w", runErr)
	}
	return out, nil
}

// --- settings --------------------------------------------------------------

// ServerInfoResult is the settings "about the server" card payload.
type ServerInfoResult struct {
	Host     string `json:"host"`
	Port     string `json:"port"`     // WG UDP port ("" if unknown)
	PanelURL string `json:"panelUrl"` // web panel URL
	Version  string `json:"version"`  // AmneziaWG version ("" if unknown)
	Uptime   string `json:"uptime"`   // localized, "" if unknown
	Clients  int    `json:"clients"`  // configured peers
}

// ServerInfo gathers a snapshot of facts about the connected server for the
// settings panel (port, version, uptime, client count).
func (a *App) ServerInfo() (ServerInfoResult, error) {
	cl, t, err := a.conn()
	if err != nil {
		return ServerInfoResult{}, err
	}
	out, err := cl.Run(deploy.ServerInfoCommand(deploy.Sudo(t.User), awgIface))
	if err != nil {
		return ServerInfoResult{}, fmt.Errorf("не удалось получить данные сервера: %w", err)
	}
	info := deploy.ParseServerInfo(out)
	return ServerInfoResult{
		Host:     a.target.Host,
		Port:     info.Port,
		PanelURL: a.panelURL(),
		Version:  strings.TrimSpace(info.Version),
		Uptime:   formatUptime(int(info.UptimeSeconds), a.lang),
		Clients:  info.Peers,
	}, nil
}

// RenameServer updates the friendly label of the connected server's saved
// profile (local only — the VPS is not touched).
func (a *App) RenameServer(label string) error {
	a.mu.Lock()
	t := a.target
	connected := a.client != nil
	a.mu.Unlock()
	if !connected || t.Host == "" {
		return fmt.Errorf("нет подключения к серверу")
	}
	label = strings.TrimSpace(label)
	d := loadProfilesDisk()
	key := profileKey(t.User, t.Host)
	for _, e := range d.Profiles {
		if profileKey(e.User, e.Host) == key {
			e.Label = label
			upsertProfile(e)
			return nil
		}
	}
	upsertProfile(ProfileEntry{Host: t.Host, User: t.User, Label: label})
	return nil
}

// ChangePanelPassword rewrites the web panel's admin password on the server
// (bcrypt hash + service restart). The new password is sent over the SSH session
// via stdin, never as a command argument.
func (a *App) ChangePanelPassword(newPassword string) error {
	cl, t, err := a.conn()
	if err != nil {
		return err
	}
	if !validPanelPassword(newPassword) {
		return fmt.Errorf("слабый пароль: минимум 6 символов, строчные и заглавные буквы, цифра и спецсимвол (например Admin2@)")
	}
	out, err := cl.RunScript(deploy.ChangePanelPasswordCommand(deploy.Sudo(t.User)), newPassword+"\n", a.logWriter("panel:log"))
	if err != nil {
		return fmt.Errorf("смена пароля не удалась: %w\n%s", err, out)
	}
	return nil
}

// --- helpers ---------------------------------------------------------------

// normalizePreset clamps the obfuscation profile to a supported value.
func normalizePreset(p string) string {
	switch strings.TrimSpace(p) {
	case "desktop":
		return "desktop"
	case "plain":
		return "plain"
	default:
		return "mobile"
	}
}

// conn returns the live client and target, or an error if not connected.
func (a *App) conn() (*deploy.Client, deploy.Target, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return nil, deploy.Target{}, fmt.Errorf("нет подключения к серверу")
	}
	return a.client, a.target, nil
}

// logWriter returns an io.Writer that streams installer output to the UI as
// events under the given name.
func (a *App) logWriter(event string) *eventWriter {
	return &eventWriter{ctx: a.ctx, event: event}
}

// eventWriter forwards written bytes to the frontend as Wails runtime events.
type eventWriter struct {
	ctx   context.Context
	event string
}

func (w *eventWriter) Write(p []byte) (int, error) {
	if w.ctx != nil {
		wruntime.EventsEmit(w.ctx, w.event, string(p))
	}
	return len(p), nil
}

// buildClientResult extracts the client config from installer output and renders
// a scannable QR PNG as a data URI.
func buildClientResult(name, output string) (ClientResult, error) {
	conf, err := deploy.ExtractConfig(output)
	if err != nil {
		return ClientResult{}, fmt.Errorf("не нашёл конфиг клиента в выводе: %w", err)
	}
	return clientResultFromConfs(name, conf, extractSecondaryConfig(output)), nil
}

func extractSecondaryConfig(output string) string {
	const begin = "-----BEGIN_AWG_CONF_SECONDARY-----"
	const end = "-----END_AWG_CONF_SECONDARY-----"
	start := strings.Index(output, begin)
	if start < 0 {
		return ""
	}
	rest := output[start+len(begin):]
	finish := strings.Index(rest, end)
	if finish < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:finish]) + "\n"
}

// validPanelPassword enforces a non-trivial admin password (the panel is reachable
// over the network): at least 6 chars with a lowercase letter, an uppercase letter,
// a digit and a special character — so "123456" fails but "Admin2@" passes.
func validPanelPassword(p string) bool {
	if len(p) < 6 {
		return false
	}
	var lower, upper, digit, special bool
	for _, r := range p {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		default:
			special = true
		}
	}
	return lower && upper && digit && special
}

// clientResultFromConf wraps a client config with a scannable QR data URI.
// Low EC + a large image keeps the long AmneziaWG config QR scannable.
func clientResultFromConf(name, conf string) ClientResult {
	return clientResultFromConfs(name, conf, "")
}

func clientResultFromConfs(name, conf, alt string) ClientResult {
	res := ClientResult{Name: name, Conf: conf, ConfAlt: alt}
	if png, err := qrcode.Encode(conf, qrcode.Low, 512); err == nil {
		res.QR = "data:image/png;base64," + encodeBase64(png)
	}
	if alt != "" {
		if png, err := qrcode.Encode(alt, qrcode.Low, 512); err == nil {
			res.QRAlt = "data:image/png;base64," + encodeBase64(png)
		}
	}
	return res
}
