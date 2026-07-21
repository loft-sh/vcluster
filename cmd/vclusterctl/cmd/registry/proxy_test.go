package registry

import (
	"net/url"
	"testing"
)

func TestNewHostRewriterAddsSlashAfterPort(t *testing.T) {
	target, err := url.Parse("https://registry.example.com/")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}

	replaceHost := newHostRewriter(target, "127.0.0.1:15000")

	got := replaceHost("https://registry.example.com/v2/")
	want := "http://127.0.0.1:15000/v2/"
	if got != want {
		t.Fatalf("replaceHost() = %q, want %q", got, want)
	}
}
