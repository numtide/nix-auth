package nixconf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureNetrcFileCreatesAndIsIdempotent(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "nix.conf")
	_ = os.WriteFile(conf, []byte("experimental-features = nix-command\n"), 0o644)

	cfg, _ := New(conf)

	changed, path, err := cfg.EnsureNetrcFile()
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}

	data, _ := os.ReadFile(conf)
	if !strings.Contains(string(data), "experimental-features = nix-command") ||
		!strings.Contains(string(data), "netrc-file = "+path) {
		t.Errorf("unexpected nix.conf:\n%s", data)
	}

	if changed, _, err := cfg.EnsureNetrcFile(); err != nil || changed {
		t.Errorf("second call: changed=%v err=%v", changed, err)
	}
}

func TestEnsureNetrcFileDoesNotOverrideExisting(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "nix.conf")
	orig := "netrc-file = /etc/custom/netrc\n"
	_ = os.WriteFile(conf, []byte(orig), 0o644)

	cfg, _ := New(conf)

	changed, current, err := cfg.EnsureNetrcFile()
	if !errors.Is(err, ErrNetrcFileConflict) || changed || current != "/etc/custom/netrc" {
		t.Errorf("changed=%v current=%q err=%v", changed, current, err)
	}

	if data, _ := os.ReadFile(conf); string(data) != orig {
		t.Errorf("nix.conf was modified:\n%s", data)
	}
}

func TestEnsureNetrcFileDoesNotInlineIncludedFiles(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "nix.conf")
	_ = os.WriteFile(filepath.Join(dir, "extra.conf"), []byte("cores = 4\n"), 0o600)
	_ = os.WriteFile(conf, []byte("!include extra.conf\n"), 0o644)

	cfg, _ := New(conf)
	if _, _, err := cfg.EnsureNetrcFile(); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(conf)
	if strings.Contains(string(data), "cores = 4") {
		t.Errorf("included content leaked into nix.conf:\n%s", data)
	}
}

func TestEnsureNetrcFileCreatesMissingConfig(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "sub", "nix.conf")
	cfg, _ := New(conf)

	if changed, _, err := cfg.EnsureNetrcFile(); err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
}
