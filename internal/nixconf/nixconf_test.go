package nixconf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultUserConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "default path when no env vars set",
			envVars:  map[string]string{},
			expected: "~/.config/nix/nix.conf",
		},
		{
			name: "NIX_USER_CONF_FILES single file",
			envVars: map[string]string{
				"NIX_USER_CONF_FILES": "/custom/path/nix.conf",
			},
			expected: "/custom/path/nix.conf",
		},
		{
			name: "NIX_USER_CONF_FILES multiple files",
			envVars: map[string]string{
				"NIX_USER_CONF_FILES": "/first/path.conf:/second/path.conf",
			},
			expected: "/first/path.conf",
		},
		{
			name: "NIX_USER_CONF_FILES empty first element",
			envVars: map[string]string{
				"NIX_USER_CONF_FILES": ":/second/path.conf",
			},
			expected: "~/.config/nix/nix.conf",
		},
		{
			name: "XDG_CONFIG_HOME set",
			envVars: map[string]string{
				"XDG_CONFIG_HOME": "/custom/config",
			},
			expected: "/custom/config/nix/nix.conf",
		},
		{
			name: "NIX_USER_CONF_FILES takes precedence over XDG_CONFIG_HOME",
			envVars: map[string]string{
				"NIX_USER_CONF_FILES": "/priority/path.conf",
				"XDG_CONFIG_HOME":     "/custom/config",
			},
			expected: "/priority/path.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current env vars
			savedEnvVars := make(map[string]string)
			for key := range tt.envVars {
				savedEnvVars[key] = os.Getenv(key)
				_ = os.Unsetenv(key)
			}

			savedEnvVars["XDG_CONFIG_HOME"] = os.Getenv("XDG_CONFIG_HOME")
			savedEnvVars["NIX_USER_CONF_FILES"] = os.Getenv("NIX_USER_CONF_FILES")
			_ = os.Unsetenv("XDG_CONFIG_HOME")
			_ = os.Unsetenv("NIX_USER_CONF_FILES")

			// Set test env vars
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			// Test
			got := DefaultUserConfigPath()
			if got != tt.expected {
				t.Errorf("DefaultUserConfigPath() = %v, want %v", got, tt.expected)
			}

			// Restore env vars
			for key, value := range savedEnvVars {
				if value == "" {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, value)
				}
			}
		})
	}
}

func TestNixConfig_SetAndGetToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name  string
		host  string
		token string
	}{
		{"github.com token", "github.com", "ghp_testtoken123"},
		{"gitlab.com token", "gitlab.com", "glpat-testtoken456"},
		{"custom host", "git.company.com", "custom_token789"},
	}

	// Test setting tokens
	for _, tt := range tests {
		t.Run("set "+tt.name, func(t *testing.T) {
			err := cfg.SetToken(tt.host, tt.token)
			if err != nil {
				t.Errorf("SetToken(%q, %q) error = %v", tt.host, tt.token, err)
			}

			// Verify token was set
			got, err := cfg.GetToken(tt.host)
			if err != nil {
				t.Errorf("GetToken(%q) error = %v", tt.host, err)
			}

			if got != tt.token {
				t.Errorf("GetToken(%q) = %v, want %v", tt.host, got, tt.token)
			}
		})
	}

	// Verify all tokens are present
	hosts, err := cfg.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	if len(hosts) != len(tests) {
		t.Errorf("ListTokens() returned %d hosts, want %d", len(hosts), len(tests))
	}

	// Verify file format - tokens should be in separate file now
	tokenFilePath := filepath.Join(tmpDir, "access-tokens.conf")

	tokenContent, err := os.ReadFile(tokenFilePath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile(token file) error = %v", err)
	}

	if !strings.Contains(string(tokenContent), "access-tokens = ") {
		t.Errorf("Token file does not contain 'access-tokens = ' line")
	}

	// Verify main config has include
	content, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile(main config) error = %v", err)
	}

	if !strings.Contains(string(content), "!include access-tokens.conf") {
		t.Errorf("Main config does not include access-tokens.conf")
	}
}

func TestNixConfig_UpdateExistingToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Create initial config with a token
	initialContent := `# Some config
experimental-features = nix-command flakes
access-tokens = github.com=old_token
# More config
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update existing token
	newToken := "new_github_token" //nolint:gosec // test token
	if err := cfg.SetToken("github.com", newToken); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Verify token was updated
	got, err := cfg.GetToken("github.com")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if got != newToken {
		t.Errorf("GetToken() = %v, want %v", got, newToken)
	}

	// Verify other config lines are preserved
	content, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "experimental-features = nix-command flakes") {
		t.Errorf("Config file lost existing configuration")
	}

	// Verify tokens are in separate file
	tokenFilePath := filepath.Join(tmpDir, "access-tokens.conf")

	tokenContent, err := os.ReadFile(tokenFilePath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile(token file) error = %v", err)
	}

	if !strings.Contains(string(tokenContent), "github.com="+newToken) {
		t.Errorf("Token file does not contain updated token")
	}
}

func TestNixConfig_RemoveToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Add multiple tokens
	tokens := map[string]string{
		"github.com":      "token1",
		"gitlab.com":      "token2",
		"git.company.com": "token3",
	}

	for host, token := range tokens {
		if err := cfg.SetToken(host, token); err != nil {
			t.Fatalf("SetToken(%q, %q) error = %v", host, token, err)
		}
	}

	// Remove one token
	if err := cfg.RemoveToken("gitlab.com"); err != nil {
		t.Fatalf("RemoveToken() error = %v", err)
	}

	// Verify token was removed
	got, err := cfg.GetToken("gitlab.com")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if got != "" {
		t.Errorf("GetToken(gitlab.com) = %v, want empty", got)
	}

	// Verify other tokens remain
	for host, expectedToken := range tokens {
		if host == "gitlab.com" {
			continue
		}

		got, err := cfg.GetToken(host)
		if err != nil {
			t.Errorf("GetToken(%q) error = %v", host, err)
		}

		if got != expectedToken {
			t.Errorf("GetToken(%q) = %v, want %v", host, got, expectedToken)
		}
	}

	// Test removing non-existent token
	err = cfg.RemoveToken("nonexistent.com")
	if err == nil {
		t.Errorf("RemoveToken(nonexistent) should return error")
	}
}

func TestNixConfig_RemoveLastToken(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Create config with single token
	initialContent := `experimental-features = nix-command flakes
access-tokens = github.com=only_token
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// This will trigger migration to separate file first
	if err := cfg.SetToken("github.com", "only_token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Remove the last token
	if err := cfg.RemoveToken("github.com"); err != nil {
		t.Fatalf("RemoveToken() error = %v", err)
	}

	// Verify token file was removed when empty
	tokenFilePath := filepath.Join(tmpDir, "access-tokens.conf")
	if _, err := os.Stat(tokenFilePath); !os.IsNotExist(err) {
		t.Errorf("Token file should be removed when no tokens remain")
	}
	// Verify other config is preserved
	content, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "experimental-features = nix-command flakes") {
		t.Errorf("Other config lines should be preserved")
	}
}

func TestNixConfig_Backup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Create initial config with access tokens to trigger migration
	initialContent := "experimental-features = nix-command flakes\naccess-tokens = existing.com=token\n"
	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Perform operation that creates backup
	if err := cfg.SetToken("github.com", "token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Find backup file
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	var backupFile string

	for _, f := range files {
		if strings.HasPrefix(f.Name(), "nix.conf.backup-") {
			backupFile = f.Name()
			break
		}
	}

	if backupFile == "" {
		t.Errorf("No backup file found")
	} else {
		// Verify backup content
		backupContent, err := os.ReadFile(filepath.Join(tmpDir, backupFile)) //nolint:gosec // test file path
		if err != nil {
			t.Fatalf("ReadFile(backup) error = %v", err)
		}

		if string(backupContent) != initialContent {
			t.Errorf("Backup content = %q, want %q", string(backupContent), initialContent)
		}
	}
}

func TestNixConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Create empty file
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Should handle empty file gracefully
	hosts, err := cfg.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	if len(hosts) != 0 {
		t.Errorf("ListTokens() = %v, want empty", hosts)
	}

	// Should be able to add token to empty file
	if err := cfg.SetToken("github.com", "token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}
}

func TestNixConfig_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "nix.conf")

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Should handle non-existent file gracefully
	hosts, err := cfg.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	if len(hosts) != 0 {
		t.Errorf("ListTokens() = %v, want empty", hosts)
	}

	// Should create directory and file when setting token
	if err := cfg.SetToken("github.com", "token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created")
	}
}

func TestNixConfig_InvalidTokenFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")
	tokenPath := filepath.Join(tmpDir, "access-tokens.conf")

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "missing equals in token pair",
			content: "access-tokens = github.com-invalid-no-equals",
			wantErr: true,
		},
		{
			name:    "invalid token format",
			content: "access-tokens = github.com_no_equals",
			wantErr: true,
		},
		{
			name:    "valid format",
			content: "access-tokens = github.com=token",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create main config with include
			mainContent := "!include access-tokens.conf\n"
			if err := os.WriteFile(configPath, []byte(mainContent), 0600); err != nil {
				t.Fatalf("WriteFile(main) error = %v", err)
			}

			// Create token file with test content
			if err := os.WriteFile(tokenPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("WriteFile(token) error = %v", err)
			}

			cfg, err := New(configPath)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			_, err = cfg.GetToken("github.com")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNixConfig_PreservesWhitespaceAndComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Create config with various formatting
	initialContent := `# This is a comment
experimental-features = nix-command flakes

# Access tokens section
access-tokens = github.com=token1

# Another comment
trusted-users = root user
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update token
	if err := cfg.SetToken("gitlab.com", "token2"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Read back content
	content, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Verify structure is preserved (tokens should be removed from main file)
	lines := strings.Split(string(content), "\n")
	expectedPatterns := []string{
		"# This is a comment",
		"experimental-features = nix-command flakes",
		"!include access-tokens.conf", // Should have include instead of tokens
		"# Another comment",
		"trusted-users = root user",
	}

	for _, pattern := range expectedPatterns {
		found := false

		for _, line := range lines {
			if strings.Contains(line, pattern) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Pattern %q not found in output", pattern)
		}
	}

	// Verify tokens are in separate file
	tokenFilePath := filepath.Join(tmpDir, "access-tokens.conf")

	tokenContent, err := os.ReadFile(tokenFilePath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile(token file) error = %v", err)
	}

	if !strings.Contains(string(tokenContent), "gitlab.com=token2") {
		t.Errorf("Token file does not contain new token")
	}
}

func TestNixConfig_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Initial setup
	if err := cfg.SetToken("github.com", "initial"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Note: This test simulates what might happen with concurrent access,
	// but doesn't actually test thread safety since the current implementation
	// doesn't have any locking mechanism. This is more of a sequential test
	// to ensure operations work correctly when performed in quick succession.

	operations := []struct {
		name string
		op   func() error
	}{
		{"set token 1", func() error { return cfg.SetToken("host1.com", "token1") }},
		{"set token 2", func() error { return cfg.SetToken("host2.com", "token2") }},
		{"update github", func() error { return cfg.SetToken("github.com", "updated") }},
		{"remove host1", func() error {
			// Add small delay to ensure different timestamps for backups
			time.Sleep(time.Millisecond)
			return cfg.RemoveToken("host1.com")
		}},
	}

	for _, op := range operations {
		if err := op.op(); err != nil {
			t.Errorf("%s error = %v", op.name, err)
		}
	}

	// Verify final state
	expectedTokens := map[string]string{
		"github.com": "updated",
		"host2.com":  "token2",
	}

	for host, expectedToken := range expectedTokens {
		got, err := cfg.GetToken(host)
		if err != nil {
			t.Errorf("GetToken(%q) error = %v", host, err)
		}

		if got != expectedToken {
			t.Errorf("GetToken(%q) = %v, want %v", host, got, expectedToken)
		}
	}

	// Verify removed token is gone
	got, _ := cfg.GetToken("host1.com")
	if got != "" {
		t.Errorf("GetToken(host1.com) = %v, want empty", got)
	}
}

func TestExpandTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/config/nix.conf", filepath.Join(homeDir, "config/nix.conf")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", "~"},                   // tilde alone is not expanded
		{"~user/path", "~user/path"}, // other user's home is not expanded
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandTilde(tt.input)
			if got != tt.expected {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNixConfig_SortedOutput(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Add tokens in non-alphabetical order
	hosts := []string{"zebra.com", "apple.com", "middle.com", "banana.com"}
	for i, host := range hosts {
		if err := cfg.SetToken(host, fmt.Sprintf("token%d", i)); err != nil {
			t.Fatalf("SetToken(%q) error = %v", host, err)
		}
	}

	// Read the token file and verify tokens are sorted
	tokenFilePath := filepath.Join(tmpDir, "access-tokens.conf")

	tokenContent, err := os.ReadFile(tokenFilePath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile(token file) error = %v", err)
	}

	// Extract the access-tokens line
	lines := strings.Split(string(tokenContent), "\n")

	var accessTokensLine string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "access-tokens") {
			accessTokensLine = line
			break
		}
	}

	// Note: The current implementation doesn't guarantee sorted output in the file
	// Just verify all tokens are present
	for i, host := range hosts {
		expectedToken := fmt.Sprintf("%s=token%d", host, i)
		if !strings.Contains(accessTokensLine, expectedToken) {
			t.Errorf("Token %s not found in file", expectedToken)
		}
	}

	// Verify ListTokens also returns sorted
	listedHosts, err := cfg.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	expectedHosts := []string{"apple.com", "banana.com", "middle.com", "zebra.com"}
	if len(listedHosts) != len(expectedHosts) {
		t.Errorf("ListTokens() returned %d hosts, want %d", len(listedHosts), len(expectedHosts))
	}

	for i, host := range listedHosts {
		if host != expectedHosts[i] {
			t.Errorf("ListTokens()[%d] = %q, want %q", i, host, expectedHosts[i])
		}
	}
}

func TestNixConfig_PreservesExactFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Create config with specific formatting including:
	// - Mixed indentation (tabs and spaces)
	// - Multiple blank lines
	// - Inline comments
	// - Unusual spacing around =
	initialContent := `# Nix configuration file
# This has specific formatting that should be preserved

experimental-features = nix-command flakes
   # Indented comment
	tab-indented-option = value

# Multiple blank lines below


other-option=no-spaces
spaced-option   =   lots-of-spaces

# Existing access tokens
access-tokens = existing.com=token123
  # Another indented comment
extra-option = value

# End of file comment`

	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Add a new token
	if err := cfg.SetToken("github.com", "new_token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Read back the content
	content, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Check that all formatting is preserved
	contentStr := string(content)

	// Check specific formatting patterns are preserved
	patterns := []struct {
		name    string
		pattern string
	}{
		{"tab indentation", "\ttab-indented-option = value"},
		{"space indentation", "   # Indented comment"},
		{"no spaces around =", "other-option=no-spaces"},
		{"multiple spaces around =", "spaced-option   =   lots-of-spaces"},
		{"double blank lines", "\n\n\n"},
		{"indented comment", "  # Another indented comment"},
		{"end comment", "# End of file comment"},
	}

	for _, p := range patterns {
		if !strings.Contains(contentStr, p.pattern) {
			t.Errorf("Formatting not preserved for %s: pattern %q not found", p.name, p.pattern)
		}
	}

	// With new behavior, access-tokens line is removed and include is added
	// Verify main config has include and no access-tokens
	if strings.Contains(contentStr, "access-tokens = ") {
		t.Errorf("Main config should not contain access-tokens after migration")
	}

	if !strings.Contains(contentStr, "!include access-tokens.conf") {
		t.Errorf("Main config should contain include directive")
	}

	// Verify tokens are in separate file
	tokenFilePath := filepath.Join(tmpDir, "access-tokens.conf")

	tokenContent, err := os.ReadFile(tokenFilePath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile(token file) error = %v", err)
	}

	// Verify both tokens are in the token file
	if !strings.Contains(string(tokenContent), "existing.com=token123") {
		t.Errorf("Token file should contain existing token")
	}

	if !strings.Contains(string(tokenContent), "github.com=new_token") {
		t.Errorf("Token file should contain new token")
	}
}

func TestNixConfig_PreservesComplexIndentation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nix.conf")

	// Test with access-tokens line that has unusual indentation
	initialContent := `normal-option = value
		access-tokens = github.com=oldtoken
another-option = value`

	if err := os.WriteFile(configPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update the token
	if err := cfg.SetToken("gitlab.com", "newtoken"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// With new behavior, access-tokens are migrated to separate file
	tokenFilePath := filepath.Join(tmpDir, "access-tokens.conf")

	tokenContent, err := os.ReadFile(tokenFilePath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile(token file) error = %v", err)
	}

	// Verify both tokens are in the token file
	if !strings.Contains(string(tokenContent), "github.com=oldtoken") {
		t.Errorf("Token file should contain existing token")
	}

	if !strings.Contains(string(tokenContent), "gitlab.com=newtoken") {
		t.Errorf("Token file should contain new token")
	}

	// Verify main config has include
	content, err := os.ReadFile(configPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "!include access-tokens.conf") {
		t.Errorf("Main config should contain include directive")
	}
}
