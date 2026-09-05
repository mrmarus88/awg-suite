package main

import "testing"

func TestValidPanelPassword(t *testing.T) {
	cases := []struct {
		name string
		pw   string
		want bool
	}{
		{"rejects all-digits", "123456", false},
		{"rejects too short even if complex", "Aa1@", false},
		{"accepts the documented example Admin2@", "Admin2@", true},
		{"rejects missing uppercase", "admin2@x", false},
		{"rejects missing lowercase", "ADMIN2@X", false},
		{"rejects missing digit", "Admin@@x", false},
		{"rejects missing special", "Admin234", false},
		{"accepts a long complex password", "Sup3r$ecret!", true},
		{"rejects empty", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := validPanelPassword(c.pw); got != c.want {
				t.Errorf("validPanelPassword(%q) = %v, want %v", c.pw, got, c.want)
			}
		})
	}
}

func TestValidEndpointHost(t *testing.T) {
	for _, endpoint := range []string{"89.124.101.33", "146.103.102.101", "vpn.example.org", ""} {
		if !validEndpointHost(endpoint) {
			t.Errorf("validEndpointHost(%q) = false, want true", endpoint)
		}
	}
	for _, endpoint := range []string{"bad host", "host:41399", "host\nEndpoint=x", "https://example.org"} {
		if validEndpointHost(endpoint) {
			t.Errorf("validEndpointHost(%q) = true, want false", endpoint)
		}
	}
}

func TestExtractSecondaryConfig(t *testing.T) {
	output := "log\n-----BEGIN_AWG_CONF-----\nEndpoint = 89.124.101.33:41399\n-----END_AWG_CONF-----\n" +
		"-----BEGIN_AWG_CONF_SECONDARY-----\nEndpoint = 146.103.102.101:41399\n-----END_AWG_CONF_SECONDARY-----\n"
	got := extractSecondaryConfig(output)
	if got != "Endpoint = 146.103.102.101:41399\n" {
		t.Errorf("extractSecondaryConfig = %q", got)
	}
}
