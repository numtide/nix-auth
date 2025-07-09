package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/numtide/nix-auth/internal/provider"
)

func TestRunStatus(t *testing.T) {
	// Save original values
	originalConfigPath := configPath
	originalRegistry := provider.GetRegistry()

	defer func() {
		configPath = originalConfigPath
		provider.SetRegistry(originalRegistry)
	}()

	tests := []struct {
		name           string
		setupConfig    func(t *testing.T) string
		setupProviders func()
		expectedOutput []string
		expectError    bool
	}{
		{
			name: "no tokens configured",
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				// Create empty config file
				err := os.WriteFile(configFile, []byte(""), 0644)
				if err != nil {
					t.Fatalf("failed to create config file: %v", err)
				}
				return configFile
			},
			setupProviders: func() {},
			expectedOutput: []string{
				"No access tokens configured.",
				"Run 'nix-auth login' to add a token.",
			},
			expectError: false,
		},
		{
			name: "single github token valid",
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := `access-tokens = github.com=gho_testtoken123
`
				err := os.WriteFile(configFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("failed to create config file: %v", err)
				}
				return configFile
			},
			setupProviders: func() {
				// Clear registry and add mock provider
				provider.SetRegistry(make(map[string]*provider.ProviderRegistration))
				provider.RegisterProvider("github", provider.ProviderRegistration{
					New: func(cfg provider.ProviderConfig) provider.Provider {
						return &mockStatusProvider{
							name:     "github",
							host:     cfg.Host,
							valid:    true,
							scopes:   []string{"repo", "read:user"},
							username: "testuser",
							fullName: "Test User",
						}
					},
					Detect: func(ctx context.Context, client *http.Client, host string) (provider.Provider, error) {
						if host == "github.com" {
							return &mockStatusProvider{
								name:     "github",
								host:     host,
								valid:    true,
								scopes:   []string{"repo", "read:user"},
								username: "testuser",
								fullName: "Test User",
							}, nil
						}
						return nil, nil
					},
				})
			},
			expectedOutput: []string{
				"Access Tokens (1 configured",
				"github.com",
				"Provider  github",
				"User      testuser (Test User)",
				"Token     gho_****n123",
				"Scopes    repo, read:user",
				"Status    ✓ Valid",
			},
			expectError: false,
		},
		{
			name: "multiple tokens with one invalid",
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := `access-tokens = github.com=gho_validtoken gitlab.com=glpat_invalidtoken
`
				err := os.WriteFile(configFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("failed to create config file: %v", err)
				}
				return configFile
			},
			setupProviders: func() {
				provider.SetRegistry(make(map[string]*provider.ProviderRegistration))
				provider.RegisterProvider("github", provider.ProviderRegistration{
					New: func(cfg provider.ProviderConfig) provider.Provider {
						return &mockStatusProvider{
							name:     "github",
							host:     cfg.Host,
							valid:    true,
							scopes:   []string{"repo"},
							username: "ghuser",
							fullName: "GitHub User",
						}
					},
					Detect: func(ctx context.Context, client *http.Client, host string) (provider.Provider, error) {
						if host == "github.com" {
							return &mockStatusProvider{
								name:     "github",
								host:     host,
								valid:    true,
								scopes:   []string{"repo"},
								username: "ghuser",
								fullName: "GitHub User",
							}, nil
						}
						return nil, nil
					},
				})
				provider.RegisterProvider("gitlab", provider.ProviderRegistration{
					New: func(cfg provider.ProviderConfig) provider.Provider {
						return &mockStatusProvider{
							name:       "gitlab",
							host:       cfg.Host,
							valid:      false,
							validError: fmt.Errorf("401 Unauthorized"),
							scopes:     []string{},
							username:   "",
							fullName:   "",
						}
					},
					Detect: func(ctx context.Context, client *http.Client, host string) (provider.Provider, error) {
						if host == "gitlab.com" {
							return &mockStatusProvider{
								name:       "gitlab",
								host:       host,
								valid:      false,
								validError: fmt.Errorf("401 Unauthorized"),
								scopes:     []string{},
								username:   "",
								fullName:   "",
							}, nil
						}
						return nil, nil
					},
				})
			},
			expectedOutput: []string{
				"Access Tokens (2 configured",
				"github.com",
				"Provider  github",
				"User      ghuser (GitHub User)",
				"Token     gho_****oken",
				"Status    ✓ Valid",
				"gitlab.com",
				"Provider  gitlab",
				"Token     glpa****oken",
				"Status    ✗ Invalid - 401 Unauthorized",
			},
			expectError: false,
		},
		{
			name: "unknown provider",
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := `access-tokens = unknown.host.com=token1234567890
`
				err := os.WriteFile(configFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("failed to create config file: %v", err)
				}
				return configFile
			},
			setupProviders: func() {
				provider.SetRegistry(make(map[string]*provider.ProviderRegistration))
				// No need to register unknown provider - it's handled internally
			},
			expectedOutput: []string{
				"unknown.host.com",
				"Provider  unknown",
				"Status    ⚠ Unknown (unverified)",
				"Token     toke****7890",
				"Scopes    None",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			configPath = tt.setupConfig(t)

			// Setup providers
			tt.setupProviders()

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			err := runStatus(nil, []string{})

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			// Check error
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Check output contains expected strings
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("output missing expected string %q\nGot output:\n%s", expected, output)
				}
			}
		})
	}
}

// mockStatusProvider implements Provider interface for status command testing
type mockStatusProvider struct {
	name       string
	host       string
	clientID   string
	valid      bool
	validError error
	scopes     []string
	username   string
	fullName   string
}

func (m *mockStatusProvider) Name() string { return m.name }
func (m *mockStatusProvider) Host() string { return m.host }

func (m *mockStatusProvider) Authenticate(ctx context.Context) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *mockStatusProvider) ValidateToken(ctx context.Context, token string) (provider.ValidationStatus, error) {
	if m.valid {
		return provider.ValidationStatusValid, nil
	}
	return provider.ValidationStatusInvalid, m.validError
}

func (m *mockStatusProvider) GetScopes() []string {
	return m.scopes
}

func (m *mockStatusProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	if !m.valid {
		return nil, fmt.Errorf("invalid token")
	}
	return m.scopes, nil
}

func (m *mockStatusProvider) GetUserInfo(ctx context.Context, token string) (string, string, error) {
	if !m.valid {
		return "", "", fmt.Errorf("invalid token")
	}
	return m.username, m.fullName, nil
}

// mockUnknownProvider implements Provider interface for unknown hosts
type mockUnknownProvider struct {
	host string
}

func (m *mockUnknownProvider) Name() string { return "unknown" }
func (m *mockUnknownProvider) Host() string { return m.host }

func (m *mockUnknownProvider) Authenticate(ctx context.Context) (string, error) {
	return "", fmt.Errorf("authentication not supported for unknown provider")
}

func (m *mockUnknownProvider) ValidateToken(ctx context.Context, token string) (provider.ValidationStatus, error) {
	// Unknown providers cannot validate tokens
	return provider.ValidationStatusUnknown, nil
}

func (m *mockUnknownProvider) GetScopes() []string {
	return []string{}
}

func (m *mockUnknownProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	// Return empty scopes for unknown providers
	return []string{}, nil
}

func (m *mockUnknownProvider) GetUserInfo(ctx context.Context, token string) (string, string, error) {
	return "", "", fmt.Errorf("user info not available for unknown provider")
}

func TestStatusCommandIntegration(t *testing.T) {
	// Test that the status command is properly registered
	if statusCmd == nil {
		t.Error("statusCmd should not be nil")
	}

	if statusCmd.Use != "status" {
		t.Errorf("expected Use to be 'status', got %q", statusCmd.Use)
	}

	if statusCmd.RunE == nil {
		t.Error("statusCmd.RunE should not be nil")
	}
}
