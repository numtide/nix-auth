package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/numtide/nix-auth/internal/nixconf"
)

func TestSetBasicAuthWritesOnlyNetrc(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := nixconf.New(filepath.Join(dir, "nix.conf"))

	old := setTokenUser
	setTokenUser = "alice"

	t.Cleanup(func() { setTokenUser = old })

	if err := setBasicAuth(cfg, "cache.example.com", []string{"cache.example.com", "made-up-password"}); err != nil {
		t.Fatal(err)
	}

	netrcData, err := os.ReadFile(filepath.Join(dir, "netrc"))
	if err != nil || !strings.Contains(string(netrcData), "machine cache.example.com login alice password made-up-password") {
		t.Fatalf("netrc = %q err=%v", netrcData, err)
	}

	if _, err := os.Stat(filepath.Join(dir, "access-tokens.conf")); !os.IsNotExist(err) {
		t.Error("access-tokens.conf must not be created for Basic Auth hosts")
	}

	conf, _ := os.ReadFile(filepath.Join(dir, "nix.conf"))
	if !strings.Contains(string(conf), "netrc-file = ") {
		t.Errorf("nix.conf missing netrc-file:\n%s", conf)
	}
}

func TestRemoveTokenCoversNetrcOnlyHosts(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := nixconf.New(filepath.Join(dir, "nix.conf"))

	if err := saveNetrc(cfg, "cache.example.com", "alice", "made-up-password"); err != nil {
		t.Fatal(err)
	}

	if err := removeToken(cfg, "cache.example.com"); err != nil {
		t.Fatalf("logout of netrc-only host failed: %v", err)
	}

	if err := removeToken(cfg, "cache.example.com"); err == nil {
		t.Error("expected error when host is in neither store")
	}
}
