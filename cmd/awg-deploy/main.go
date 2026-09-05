// Command awg-deploy installs and manages a self-hosted AmneziaWG VPN on a
// remote server over SSH — a single cross-platform binary (incl. Windows .exe).
//
// Usage:
//
//	awg-deploy install   user@host[:sshport] [flags]
//	awg-deploy add-client user@host[:sshport] <name> [flags]
//	awg-deploy monitor   user@host[:sshport] [flags]
//
// The installer script is embedded, so nothing needs to be present on the
// server beforehand — awg-deploy pipes it over SSH and runs it non-interactively.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"

	amneziawg "github.com/hennessyxo/awg-suite"
	"github.com/hennessyxo/awg-suite/internal/awg"
	"github.com/hennessyxo/awg-suite/internal/deploy"
	"github.com/hennessyxo/awg-suite/internal/ui"
)

// stdin is a single shared reader so successive line prompts don't drop input.
var stdin = bufio.NewReader(os.Stdin)

func main() {
	// No subcommand → friendly interactive wizard (e.g. double-clicked on macOS).
	if len(os.Args) < 2 {
		if err := runWizard(); err != nil {
			fmt.Fprintln(os.Stderr, "awg-deploy:", err)
			os.Exit(1)
		}
		return
	}
	var err error
	switch os.Args[1] {
	case "install":
		err = runInstall(os.Args[2:])
	case "add-client":
		err = runAddClient(os.Args[2:])
	case "set-secondary-endpoint":
		err = runSetSecondaryEndpoint(os.Args[2:])
	case "remove-client":
		err = runRemoveClient(os.Args[2:])
	case "list":
		err = runList(os.Args[2:])
	case "menu":
		err = runMenu(os.Args[2:])
	case "uninstall":
		err = runUninstall(os.Args[2:])
	case "monitor":
		err = runMonitor(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "awg-deploy:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`awg-deploy — установка и управление AmneziaWG по SSH

  awg-deploy install       user@host[:port] [--client phone]   # port auto-picked, universal profile
  awg-deploy add-client    user@host[:port] <name>
	  awg-deploy set-secondary-endpoint user@host[:port] <IP-or-host>
  awg-deploy remove-client user@host[:port] <name>
  awg-deploy list          user@host[:port]
  awg-deploy menu          user@host[:port]            # interactive menu over SSH
  awg-deploy monitor       user@host[:port] [--iface awg0] [--interval 2s]
  awg-deploy uninstall     user@host[:port]            # remove everything (asks to confirm)

Аутентификация: --identity <ключ> или пароль (спросит). Общие флаги: --identity,
--known-hosts, --accept-new.
`)
}

// runWizard is the friendly default flow: ask for the server and password, then
// connect and run the installer/menu directly ON the server (everything happens
// server-side; this tool is just the SSH launcher).
func runWizard() error {
	fmt.Println("=== AmneziaWG ===")
	fmt.Println("Подключусь к твоему серверу по SSH и запущу установку и меню прямо на нём.")
	fmt.Println()

	raw := promptLine("IP-адрес сервера (или user@IP; по умолчанию пользователь root): ")
	if raw == "" {
		return errors.New("адрес сервера не указан")
	}
	t, err := deploy.ParseTarget(raw)
	if err != nil {
		return err
	}

	id, kh, accept := "", defaultKnownHosts(), true
	cl, err := connect(t, authFlags{identity: &id, knownHosts: &kh, acceptNew: &accept})
	if err != nil {
		return err
	}
	defer cl.Close()
	return serverMenu(cl, t)
}

// serverMenu uploads the installer to the server and runs it interactively over
// an SSH PTY — install (if needed) and the management menu all run server-side.
func serverMenu(cl *deploy.Client, t deploy.Target) error {
	const remote = "/tmp/awg-install.sh"
	if err := cl.WriteFile(remote, amneziawg.InstallerScript); err != nil {
		return fmt.Errorf("не удалось загрузить скрипт на сервер: %w", err)
	}
	fmt.Println("→ Подключаюсь к серверу...")
	return cl.Interactive(fmt.Sprintf("%sbash %s; rm -f %s", deploy.Sudo(t.User), remote, remote))
}

// authFlags holds the SSH auth/host-key flags shared by all subcommands.
type authFlags struct {
	identity   *string
	knownHosts *string
	acceptNew  *bool
}

func registerAuthFlags(fs *flag.FlagSet) authFlags {
	return authFlags{
		identity:   fs.String("identity", "", "SSH private key file (else password prompt)"),
		knownHosts: fs.String("known-hosts", defaultKnownHosts(), "known_hosts path"),
		acceptNew:  fs.Bool("accept-new", false, "trust an unknown host key without prompting"),
	}
}

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	af := registerAuthFlags(fs)
	port := fs.String("port", "", "AmneziaWG UDP port (default: random on server)")
	preset := fs.String("preset", "mobile", "obfuscation preset (mobile works on phone and PC)")
	client := fs.String("client", "phone", "first client name")
	serverIP := fs.String("server-ip", "", "public IP/host clients connect to (default: autodetect)")
	serverIPAlt := fs.String("server-ip-alt", "", "optional fallback public IP/host for second client profile")
	dns1 := fs.String("dns1", "", "client DNS 1 (default 1.1.1.1)")
	dns2 := fs.String("dns2", "", "client DNS 2 (default 1.0.0.1)")
	out := fs.String("out", "", "where to save the client .conf (default <client>.conf)")
	_ = fs.Parse(args)

	t, err := targetArg(fs)
	if err != nil {
		return err
	}
	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()

	env := map[string]string{"AWG_PRESET": *preset, "AWG_CLIENT": *client}
	putIf(env, "AWG_PORT", *port)
	putIf(env, "AWG_SERVER_IP", *serverIP)
	putIf(env, "AWG_SERVER_IP_ALT", *serverIPAlt)
	putIf(env, "AWG_DNS1", *dns1)
	putIf(env, "AWG_DNS2", *dns2)

	fmt.Printf("→ Устанавливаю AmneziaWG на %s ...\n\n", t.Addr())
	output, err := cl.RunScript(deploy.InstallCommand(deploy.Sudo(t.User), env), amneziawg.InstallerScript, os.Stdout)
	if err != nil {
		return fmt.Errorf("установка не удалась: %w", err)
	}
	if deploy.AlreadyInstalled(output) {
		fmt.Printf("\nℹ Сервер уже настроен. Дальше управляй так:\n"+
			"  awg-deploy add-client    %s <имя>   — добавить клиента\n"+
			"  awg-deploy list          %s         — список клиентов\n"+
			"  awg-deploy remove-client %s <имя>   — удалить клиента\n"+
			"  awg-deploy monitor       %s         — живой мониторинг\n",
			t.Addr(), t.Addr(), t.Addr(), t.Addr())
		return nil
	}
	return saveAndShow(output, defaultOut(*out, *client))
}

func runRemoveClient(args []string) error {
	fs := flag.NewFlagSet("remove-client", flag.ExitOnError)
	af := registerAuthFlags(fs)
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) < 2 {
		return errors.New("usage: awg-deploy remove-client user@host <name>")
	}
	t, err := deploy.ParseTarget(rest[0])
	if err != nil {
		return err
	}
	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()
	out, err := cl.RunScript(deploy.RemoveClientCommand(deploy.Sudo(t.User), rest[1]), amneziawg.InstallerScript, os.Stdout)
	if err != nil {
		return fmt.Errorf("удаление не удалось: %w\n%s", err, out)
	}
	return nil
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	af := registerAuthFlags(fs)
	_ = fs.Parse(args)
	t, err := targetArg(fs)
	if err != nil {
		return err
	}
	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()
	out, err := cl.RunScript(deploy.ListClientsCommand(deploy.Sudo(t.User)), amneziawg.InstallerScript, os.Stdout)
	if err != nil {
		return fmt.Errorf("не удалось получить список: %w\n%s", err, out)
	}
	return nil
}

func runAddClient(args []string) error {
	fs := flag.NewFlagSet("add-client", flag.ExitOnError)
	af := registerAuthFlags(fs)
	out := fs.String("out", "", "where to save the client .conf (default <name>.conf)")
	_ = fs.Parse(args)

	rest := fs.Args()
	if len(rest) < 2 {
		return errors.New("usage: awg-deploy add-client user@host <name>")
	}
	t, err := deploy.ParseTarget(rest[0])
	if err != nil {
		return err
	}
	name := rest[1]

	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()

	fmt.Printf("→ Создаю клиента %q на %s ...\n\n", name, t.Addr())
	output, err := cl.RunScript(deploy.AddClientCommand(deploy.Sudo(t.User), name, nil), amneziawg.InstallerScript, os.Stdout)
	if err != nil {
		return fmt.Errorf("создание клиента не удалось: %w", err)
	}
	return saveAndShow(output, defaultOut(*out, name))
}

func runSetSecondaryEndpoint(args []string) error {
	fs := flag.NewFlagSet("set-secondary-endpoint", flag.ExitOnError)
	af := registerAuthFlags(fs)
	_ = fs.Parse(args)
	rest := fs.Args()
	if len(rest) < 2 {
		return errors.New("usage: awg-deploy set-secondary-endpoint user@host <IP-or-host>")
	}
	t, err := deploy.ParseTarget(rest[0])
	if err != nil {
		return err
	}
	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()
	out, err := cl.RunScript(deploy.SetSecondaryEndpointCommand(deploy.Sudo(t.User), rest[1]), amneziawg.InstallerScript, os.Stdout)
	if err != nil {
		return fmt.Errorf("не удалось сохранить резервный Endpoint: %w\n%s", err, out)
	}
	return nil
}

// runMenu uploads the installer to the server and opens its interactive menu
// over an SSH PTY (the script isn't otherwise stored on the server).
func runMenu(args []string) error {
	fs := flag.NewFlagSet("menu", flag.ExitOnError)
	af := registerAuthFlags(fs)
	_ = fs.Parse(args)
	t, err := targetArg(fs)
	if err != nil {
		return err
	}
	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()
	return serverMenu(cl, t)
}

func runUninstall(args []string) error {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	af := registerAuthFlags(fs)
	_ = fs.Parse(args)
	t, err := targetArg(fs)
	if err != nil {
		return err
	}
	fmt.Printf("Это ПОЛНОСТЬЮ удалит AmneziaWG, веб-панель, всех клиентов и конфиги с %s.\n", t.Addr())
	if !promptYesNo("Точно удалить?") {
		fmt.Println("Отменено.")
		return nil
	}
	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()
	out, err := cl.RunScript(deploy.UninstallCommand(deploy.Sudo(t.User)), amneziawg.InstallerScript, os.Stdout)
	if err != nil {
		return fmt.Errorf("удаление не удалось: %w\n%s", err, out)
	}
	return nil
}

func runMonitor(args []string) error {
	fs := flag.NewFlagSet("monitor", flag.ExitOnError)
	af := registerAuthFlags(fs)
	iface := fs.String("iface", "awg0", "AmneziaWG interface")
	interval := fs.Duration("interval", 2*time.Second, "refresh interval")
	_ = fs.Parse(args)

	t, err := targetArg(fs)
	if err != nil {
		return err
	}
	cl, err := connect(t, af)
	if err != nil {
		return err
	}
	defer cl.Close()
	return monitorWith(cl, t, *iface, *interval)
}

// monitorWith renders the live TUI from a remote `awg show dump` over SSH.
func monitorWith(cl *deploy.Client, t deploy.Target, iface string, interval time.Duration) error {
	sudo := deploy.Sudo(t.User)
	names := map[string]string{}
	if confOut, e := cl.Run(deploy.ReadConfCommand(sudo, iface)); e == nil {
		names = awg.ParseNames(confOut)
	}
	src := sshSource{cl: cl, sudo: sudo, iface: iface, names: names}
	prog := tea.NewProgram(ui.New(src, iface, interval), tea.WithAltScreen())
	_, err := prog.Run()
	return err
}

// sshSource feeds the TUI from a remote `awg show ... dump` over SSH.
type sshSource struct {
	cl    *deploy.Client
	sudo  string
	iface string
	names map[string]string
}

func (s sshSource) Fetch() (awg.Snapshot, error) {
	out, err := s.cl.Run(deploy.MonitorDumpCommand(s.sudo, s.iface))
	if err != nil {
		return awg.Snapshot{}, fmt.Errorf("remote awg show: %w", err)
	}
	snap, err := awg.ParseDump(s.iface, out, time.Now())
	if err == nil {
		awg.ApplyNames(snap.Peers, s.names)
	}
	return snap, err
}

// --- connection helpers ----------------------------------------------------

func targetArg(fs *flag.FlagSet) (deploy.Target, error) {
	if fs.NArg() < 1 {
		return deploy.Target{}, errors.New("missing target: user@host[:port]")
	}
	return deploy.ParseTarget(fs.Arg(0))
}

func connect(t deploy.Target, af authFlags) (*deploy.Client, error) {
	auth, err := authMethods(*af.identity)
	if err != nil {
		return nil, err
	}
	hk, err := hostKeyCallback(*af.knownHosts, *af.acceptNew)
	if err != nil {
		return nil, err
	}
	return deploy.Dial(t, auth, hk, 15*time.Second)
}

func authMethods(identity string) ([]ssh.AuthMethod, error) {
	if identity != "" {
		key, err := os.ReadFile(identity)
		if err != nil {
			return nil, fmt.Errorf("reading key %s: %w", identity, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parsing key (passphrase-protected keys aren't supported; use ssh-agent): %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	pw, err := promptPassword("SSH password: ")
	if err != nil {
		return nil, err
	}
	return []ssh.AuthMethod{ssh.Password(pw)}, nil
}

// hostKeyCallback verifies against known_hosts, with trust-on-first-use for
// unknown hosts and a hard failure on a changed key (possible MITM).
func hostKeyCallback(path string, acceptNew bool) (ssh.HostKeyCallback, error) {
	if err := ensureFile(path); err != nil {
		return nil, err
	}
	base, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("loading known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := base(hostname, remote, key); err == nil {
			return nil
		} else {
			var ke *knownhosts.KeyError
			if !errors.As(err, &ke) {
				return err
			}
			if len(ke.Want) > 0 {
				return fmt.Errorf("REMOTE HOST KEY CHANGED for %s — possible MITM, refusing to connect", hostname)
			}
		}
		fp := ssh.FingerprintSHA256(key)
		if !acceptNew {
			if !promptYesNo(fmt.Sprintf("Неизвестный хост %s\n  отпечаток: %s\nДоверять и продолжить?", hostname, fp)) {
				return errors.New("host key rejected")
			}
		}
		return appendKnownHost(path, hostname, key)
	}, nil
}

func appendKnownHost(path, hostname string, key ssh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
	_, err = f.WriteString(line + "\n")
	return err
}

// --- small utilities -------------------------------------------------------

func saveAndShow(installerOutput, outPath string) error {
	conf, err := deploy.ExtractConfig(installerOutput)
	if err != nil {
		return fmt.Errorf("не нашёл конфиг клиента в выводе: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(conf), 0o600); err != nil {
		return err
	}

	// A WireGuard+obfuscation config is long, so a terminal QR ends up too large
	// to scan reliably. Write a PNG instead (scans cleanly from the screen) and
	// open it for the user.
	// Low error-correction + a large image keeps the (long) AmneziaWG config QR
	// sparse enough to scan from a phone.
	pngPath := strings.TrimSuffix(outPath, ".conf") + ".png"
	if err := qrcode.WriteFile(conf, qrcode.Low, 1024, pngPath); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not write QR image:", err)
		pngPath = ""
	}

	fmt.Printf("\n✓ Конфиг сохранён: %s\n", outPath)
	if pngPath != "" {
		fmt.Printf("✓ QR-картинка:     %s\n", pngPath)
	}
	fmt.Println("\nКак подключиться (приложение AmneziaWG):")
	fmt.Println("  • iOS — App Store: https://apps.apple.com/app/amneziawg/id6478942365")
	fmt.Println("  • Android / Windows: https://amnezia.org/downloads")
	if pngPath != "" {
		fmt.Printf("  • затем отсканируй QR %s или импортируй файл %s\n", pngPath, outPath)
		openFile(pngPath)
	} else {
		fmt.Printf("  • затем импортируй файл %s\n", outPath)
	}
	return nil
}

// openFile best-effort opens a file with the OS default app (so the QR shows up).
func openFile(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}

func putIf(m map[string]string, k, v string) {
	if strings.TrimSpace(v) != "" {
		m[k] = v
	}
}

func defaultOut(out, name string) string {
	if out != "" {
		return out
	}
	return name + ".conf"
}

func defaultKnownHosts() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "known_hosts"
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

func ensureFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func promptLine(label string) string {
	fmt.Print(label)
	line, _ := stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptYesNo(prompt string) bool {
	ans := strings.ToLower(promptLine(prompt + " [y/N]: "))
	return ans == "y" || ans == "yes"
}
