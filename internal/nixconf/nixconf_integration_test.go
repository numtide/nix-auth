package nixconf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests verify the new behavior is working correctly through the main NixConfig interface

func TestNixConfigIntegration_SetAndGetToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		host  string
		token string
	}{
		{"github.com", "ghs_test123"},
		{"gitlab.com", "glpat_test456"},
		{"custom.example.com", "custom_token"},
	}

	// Set tokens
	for _, tt := range tests {
		t.Run("set_"+tt.host, func(t *testing.T) {
			if err := cfg.SetToken(tt.host, tt.token); err != nil {
				t.Errorf("SetToken() error = %v", err)
			}
		})
	}

	// Verify tokens can be retrieved
	for _, tt := range tests {
		t.Run("get_"+tt.host, func(t *testing.T) {
			token, err := cfg.GetToken(tt.host)
			if err != nil {
				t.Errorf("GetToken() error = %v", err)
			}

			if token != tt.token {
				t.Errorf("GetToken() = %v, want %v", token, tt.token)
			}
		})
	}

	// Verify tokens are in separate file
	tokenPath := filepath.Join(tmpDir, "access-tokens.conf")
	if _, err := os.Stat(tokenPath); err != nil {
		t.Error("access-tokens.conf not created")
	}

	// Verify main config has include directive
	mainContent, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(mainContent), "!include access-tokens.conf") {
		t.Error("Main config missing include directive")
	}

	// Verify main config does NOT have access-tokens
	if strings.Contains(string(mainContent), "access-tokens =") {
		t.Error("Main config should not contain access-tokens setting")
	}
}

func TestNixConfigIntegration_RemoveToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	cfg, err := New(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Set tokens
	tokens := map[string]string{
		"github.com": "token1",
		"gitlab.com": "token2",
		"other.com":  "token3",
	}

	for host, token := range tokens {
		if err := cfg.SetToken(host, token); err != nil {
			t.Fatal(err)
		}
	}

	// Remove one token
	if err := cfg.RemoveToken("gitlab.com"); err != nil {
		t.Errorf("RemoveToken() error = %v", err)
	}

	// Verify it's gone
	token, err := cfg.GetToken("gitlab.com")
	if err != nil {
		t.Error(err)
	}

	if token != "" {
		t.Error("Token should be removed")
	}

	// Verify others still exist
	if token, _ := cfg.GetToken("github.com"); token != "token1" {
		t.Error("Other tokens should remain")
	}
}

func TestNixConfigIntegration_MigrationFromExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Create config with tokens in main file (old style)
	oldContent := `experimental-features = nix-command flakes
access-tokens = github.com=old_token gitlab.com=old_token2
trusted-users = alice
`
	if err := os.WriteFile(configPath, []byte(oldContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatal(err)
	}

	// Add new token - should trigger migration
	if err := cfg.SetToken("new.com", "new_token"); err != nil {
		t.Fatal(err)
	}

	// Verify all tokens are accessible
	expectedTokens := map[string]string{
		"github.com": "old_token",
		"gitlab.com": "old_token2",
		"new.com":    "new_token",
	}

	for host, expectedToken := range expectedTokens {
		token, err := cfg.GetToken(host)
		if err != nil {
			t.Errorf("GetToken(%s) error = %v", host, err)
		}

		if token != expectedToken {
			t.Errorf("GetToken(%s) = %v, want %v", host, token, expectedToken)
		}
	}

	// Verify migration happened
	mainContent, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatal(err)
	}

	// Should have include
	if !strings.Contains(string(mainContent), "!include access-tokens.conf") {
		t.Error("Include directive not added")
	}

	// Should NOT have access-tokens in main file
	lines := strings.Split(string(mainContent), "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "access-tokens") && strings.Contains(line, "=") {
			t.Error("access-tokens not removed from main config")
		}
	}

	// Other settings should be preserved
	if !strings.Contains(string(mainContent), "experimental-features = nix-command flakes") {
		t.Error("Other settings not preserved")
	}

	if !strings.Contains(string(mainContent), "trusted-users = alice") {
		t.Error("Other settings not preserved")
	}
}
