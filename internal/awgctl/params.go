// Package awgctl is the control plane for the web panel: it reads the server
// state written by amneziawg-install.sh and manages clients (add/revoke). The
// file-manipulation logic is kept pure and testable; the parts that shell out
// to `awg` live in the thin Controller implementation.
package awgctl

import (
	"bufio"
	"strings"
)

// Params mirrors the key=value file at /etc/amnezia/amneziawg/params written by
// the installer. Only the fields the panel needs are kept.
type Params struct {
	ServerPubIP string
	// ServerPubIPAlt is an optional fallback client endpoint. It is absent on
	// installations made before multi-endpoint support and must never replace
	// ServerPubIP, which remains the primary/backward-compatible endpoint.
	ServerPubIPAlt string
	ServerPort     string
	ServerPubKey   string
	ClientDNS1     string
	ClientDNS2     string
	ClientMTU      string
	Jc, Jmin, Jmax string
	S1, S2         string
	H1, H2, H3, H4 string
}

// ParseParams reads the installer's params file content.
func ParseParams(content string) Params {
	kv := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return Params{
		ServerPubIP:    kv["SERVER_PUB_IP"],
		ServerPubIPAlt: kv["SERVER_PUB_IP_ALT"],
		ServerPort:     kv["SERVER_PORT"],
		ServerPubKey:   kv["SERVER_PUB_KEY"],
		ClientDNS1:     kv["CLIENT_DNS_1"],
		ClientDNS2:     kv["CLIENT_DNS_2"],
		ClientMTU:      orDefault(kv["CLIENT_MTU"], "1420"),
		Jc:             kv["JC"],
		Jmin:           kv["JMIN"],
		Jmax:           kv["JMAX"],
		S1:             kv["S1"],
		S2:             kv["S2"],
		H1:             kv["H1"],
		H2:             kv["H2"],
		H3:             kv["H3"],
		H4:             kv["H4"],
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
