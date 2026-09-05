package awgctl

import (
	"strings"
	"testing"
)

const baseConf = `[Interface]
Address = 10.66.66.1/24,fd42:42:42::1/64
ListenPort = 51820
PrivateKey = SRV=

# BEGIN_PEER phone
[Peer]
PublicKey = PH=
PresharedKey = PSK1=
AllowedIPs = 10.66.66.2/32,fd42:42:42::2/128
# END_PEER phone
`

func TestSanitizeName(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		valid bool
	}{
		{"phone", "phone", true},
		{"my phone!", "my_phone_", true},
		{"", "", false},
		{"привет", "______", true}, // 6 cyrillic runes -> 6 underscores
	}
	for _, c := range cases {
		got, ok := SanitizeName(c.in)
		if got != c.want || ok != c.valid {
			t.Errorf("SanitizeName(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.valid)
		}
	}
}

func TestListAndHasPeer(t *testing.T) {
	names := ListPeerNames(baseConf)
	if len(names) != 1 || names[0] != "phone" {
		t.Fatalf("ListPeerNames = %v, want [phone]", names)
	}
	if !HasPeer(baseConf, "phone") {
		t.Error("HasPeer(phone) = false, want true")
	}
	if HasPeer(baseConf, "laptop") {
		t.Error("HasPeer(laptop) = true, want false")
	}
}

func TestNextOctet(t *testing.T) {
	// .1 (server) and .2 (phone) are used → next is .3
	got, err := NextOctet(baseConf)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Errorf("NextOctet = %d, want 3", got)
	}
}

func TestAddAndRemovePeer_RoundTrip(t *testing.T) {
	conf := AddPeerBlock(baseConf, "laptop", "LT=", "PSK2=", 3)
	if !HasPeer(conf, "laptop") {
		t.Fatal("laptop not added")
	}
	if !strings.Contains(conf, "AllowedIPs = 10.66.66.3/32,fd42:42:42::3/128") {
		t.Error("laptop AllowedIPs missing/incorrect")
	}
	// Existing peer must be untouched.
	if !HasPeer(conf, "phone") {
		t.Error("phone disappeared after add")
	}

	conf, removed := RemovePeerBlock(conf, "laptop")
	if !removed {
		t.Error("RemovePeerBlock reported not removed")
	}
	if HasPeer(conf, "laptop") {
		t.Error("laptop still present after removal")
	}
	if !HasPeer(conf, "phone") {
		t.Error("phone removed by mistake")
	}
}

func TestRemovePeer_NotFound(t *testing.T) {
	_, removed := RemovePeerBlock(baseConf, "ghost")
	if removed {
		t.Error("removing non-existent peer reported removed=true")
	}
}

func TestParseServerClients(t *testing.T) {
	conf := AddPeerBlock(baseConf, "laptop", "LT=", "PSK2=", 7)
	clients := ParseServerClients(conf)
	if len(clients) != 2 {
		t.Fatalf("got %d clients, want 2", len(clients))
	}
	byName := map[string]ServerClient{}
	for _, c := range clients {
		byName[c.Name] = c
	}
	if p := byName["phone"]; p.PubKey != "PH=" || p.Octet != 2 {
		t.Errorf("phone parsed wrong: %+v", p)
	}
	if l := byName["laptop"]; l.PubKey != "LT=" || l.Octet != 7 {
		t.Errorf("laptop parsed wrong: %+v", l)
	}
	if !strings.Contains(byName["laptop"].Block, "# BEGIN_PEER laptop") {
		t.Error("block should include the fenced markers")
	}
}

func TestRenamePeer(t *testing.T) {
	conf := AddPeerBlock(baseConf, "laptop", "LT=", "PSK2=", 3)
	renamed := RenamePeer(conf, "laptop", "work-laptop")
	if HasPeer(renamed, "laptop") {
		t.Error("old name should be gone")
	}
	if !HasPeer(renamed, "work-laptop") {
		t.Error("new name should be present")
	}
	// The original peer and the renamed peer's keys are untouched.
	if !HasPeer(renamed, "phone") {
		t.Error("phone should be unaffected")
	}
	if !strings.Contains(renamed, "PublicKey = LT=") {
		t.Error("rename must not touch the peer's keys")
	}
	// A name that is a substring of another must not be partially matched.
	if got := RenamePeer("# BEGIN_PEER phone2\n", "phone", "x"); got != "# BEGIN_PEER phone2\n" {
		t.Errorf("partial name match: %q", got)
	}
}

func TestRenderClientConfig(t *testing.T) {
	p := Params{
		ServerPubIP: "203.0.113.7", ServerPort: "51820", ServerPubKey: "SRVPUB=",
		ClientDNS1: "1.1.1.1", ClientDNS2: "1.0.0.1", ClientMTU: "1280",
		Jc: "3", Jmin: "40", Jmax: "70", S1: "33", S2: "99",
		H1: "11", H2: "22", H3: "33", H4: "44",
	}
	cfg := RenderClientConfig(p, ClientOverrides{}, "CLIPRIV=", "PSK=", 5)
	for _, want := range []string{
		"PrivateKey = CLIPRIV=",
		"Address = 10.66.66.5/32,fd42:42:42::5/128",
		"DNS = 1.1.1.1,1.0.0.1",
		"MTU = 1280",
		"Jc = 3",
		"Endpoint = 203.0.113.7:51820",
		"AllowedIPs = 0.0.0.0/0,::/0",
		"PersistentKeepalive = 25",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("client config missing %q\n%s", want, cfg)
		}
	}
}

func TestRenderClientConfig_Overrides(t *testing.T) {
	p := Params{
		ServerPubIP: "203.0.113.7", ServerPort: "51820", ServerPubKey: "SRVPUB=",
		ClientDNS1: "1.1.1.1", ClientDNS2: "1.0.0.1", ClientMTU: "1280",
	}
	ov := ClientOverrides{
		AllowedIPs: "10.0.0.0/8,192.168.0.0/16", // split tunnel
		DNS:        "9.9.9.9",
		MTU:        "1380",
	}
	cfg := RenderClientConfig(p, ov, "CLIPRIV=", "PSK=", 5)
	for _, want := range []string{
		"AllowedIPs = 10.0.0.0/8,192.168.0.0/16",
		"DNS = 9.9.9.9",
		"MTU = 1380",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("override not applied: missing %q\n%s", want, cfg)
		}
	}
	if strings.Contains(cfg, "0.0.0.0/0") {
		t.Errorf("split tunnel should not contain full-tunnel route\n%s", cfg)
	}
}

func TestParseParams(t *testing.T) {
	content := `SERVER_PUB_IP=203.0.113.7
SERVER_PUB_IP_ALT=198.51.100.9
SERVER_PORT=51820
SERVER_PUB_KEY=SRVPUB=
CLIENT_DNS_1=1.1.1.1
CLIENT_DNS_2=1.0.0.1
JC=3
# a comment
JMIN=40
`
	p := ParseParams(content)
	if p.ServerPubIP != "203.0.113.7" || p.ServerPubIPAlt != "198.51.100.9" || p.ServerPort != "51820" {
		t.Errorf("server fields wrong: %+v", p)
	}
	if p.Jc != "3" || p.Jmin != "40" {
		t.Errorf("obfuscation fields wrong: %+v", p)
	}
	if p.ClientMTU != "1420" { // default when absent
		t.Errorf("ClientMTU default = %q, want 1420", p.ClientMTU)
	}
}

func TestWithEndpoint_OnlyReplacesEndpoint(t *testing.T) {
	primary := "[Peer]\nPublicKey = SERVER=\nEndpoint = 89.124.101.33:41399\nAllowedIPs = 0.0.0.0/0\n"
	got := WithEndpoint(primary, "146.103.102.101", "41399")
	if !strings.Contains(got, "Endpoint = 146.103.102.101:41399") {
		t.Fatalf("fallback Endpoint missing: %q", got)
	}
	if strings.Contains(got, "89.124.101.33") || !strings.Contains(got, "PublicKey = SERVER=") {
		t.Errorf("only Endpoint should change: %q", got)
	}
}
