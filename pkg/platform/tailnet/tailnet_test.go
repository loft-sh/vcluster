package tailnet

import (
	"regexp"
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := map[string]string{
		"MyLaptop":        "mylaptop",
		"my.laptop.local": "my-laptop-local",
		"Alice Smith":     "alice-smith",
		"  weird__name  ": "weird-name",
		"---":             "",
		"UPPER_CASE-123":  "upper-case-123",
		"a@b#c":           "a-b-c",
	}
	for in, want := range cases {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildClientHostname(t *testing.T) {
	labelRE := regexp.MustCompile(`^[a-z0-9-]+-[0-9a-f]{8}\.[a-z0-9-]+\.client\.slurm$`)

	h1, err := BuildClientHostname("Alice Smith")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !labelRE.MatchString(h1) {
		t.Errorf("hostname %q does not match expected shape", h1)
	}
	if !strings.HasSuffix(h1, ".alice-smith.client.slurm") {
		t.Errorf("hostname %q does not embed sanitized user", h1)
	}

	// A second call must yield a different random suffix.
	h2, err := BuildClientHostname("Alice Smith")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Errorf("expected distinct hostnames across calls, got %q twice", h1)
	}

	// Empty user falls back to "user".
	h3, err := BuildClientHostname("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(h3, ".user.client.slurm") {
		t.Errorf("hostname %q should fall back to .user.client.slurm", h3)
	}
}

func TestControlURL(t *testing.T) {
	cases := map[string]string{
		"https://my.loft.host":  "https://my.loft.host/coordinator/",
		"https://my.loft.host/": "https://my.loft.host/coordinator/",
		"http://localhost:8080": "http://localhost:8080/coordinator/",
	}
	for in, want := range cases {
		if got := controlURL(in); got != want {
			t.Errorf("controlURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPeerMatches(t *testing.T) {
	const target = "ssh.demo.myproject.slurm"
	cases := []struct {
		name     string
		hostName string
		dnsName  string
		want     bool
	}{
		{"exact hostname", target, "", true},
		{"fqdn with magicdns suffix", "", target + ".ts.net.", true},
		{"fqdn exact with trailing dot", "", target + ".", true},
		{"different instance", "ssh.other.myproject.slurm", "ssh.other.myproject.slurm.", false},
		{"prefix collision is not a match", "", "ssh.demo.myproject.slurmx.", false},
		{"unrelated", "some-client.alice.client.slurm", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := peerMatches(tc.hostName, tc.dnsName, target); got != tc.want {
				t.Errorf("peerMatches(%q, %q, %q) = %v, want %v", tc.hostName, tc.dnsName, target, got, tc.want)
			}
		})
	}
}
