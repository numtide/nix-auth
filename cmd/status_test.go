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

// captureStatusOutput captures the stdout output from running the status command.
func captureStatusOutput(t *testing.T) (string, error) {
	t.Helper()

	var buf bytes.Buffer

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	err := runStatus(nil, []string{})

	// Restore stdout
	_ = w.Close()

	os.Stdout = oldStdout

	_, _ = buf.ReadFrom(r)

	return buf.String(), err
}

// setupMockGitHubProvider sets up a mock GitHub provider.
func setupMockGitHubProvider(valid bool) {
	provider.RegisterProvider("github", provider.Registration{
		New: func(cfg provider.Config) provider.Provider {
			return &mockStatusProvider{
				name:     "github",
				host:     cfg.Host,
				valid:    valid,
				scopes:   []string{"repo", "read:user"},
				username: "testuser",
				fullName: "Test User",
			}
		},
		Detect: func(_ context.Context, _ *http.Client, host string) (provider.Provider, error) {
			if host == "github.com" {
				return &mockStatusProvider{
					name:     "github",
					host:     host,
					valid:    valid,
					scopes:   []string{"repo", "read:user"},
					username: "testuser",
					fullName: "Test User",
				}, nil
			}
			return nil, nil
		},
	})
}

// setupMockGitLabProvider sets up a mock GitLab provider.
func setupMockGitLabProvider(valid bool) {
	provider.RegisterProvider("gitlab", provider.Registration{
		New: func(cfg provider.Config) provider.Provider {
			return &mockStatusProvider{
				name:       "gitlab",
				host:       cfg.Host,
				valid:      valid,
				validError: fmt.Errorf("401 Unauthorized"),
				scopes:     []string{},
				username:   "",
				fullName:   "",
			}
		},
		Detect: func(_ context.Context, _ *http.Client, host string) (provider.Provider, error) {
			if host == "gitlab.com" {
				return &mockStatusProvider{
					name:       "gitlab",
					host:       host,
					valid:      valid,
					validError: fmt.Errorf("401 Unauthorized"),
					scopes:     []string{},
					username:   "",
					fullName:   "",
				}, nil
			}
			return nil, nil
		},
	})
}

// createTestConfig creates a test configuration file with the given content.
func createTestConfig(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "nix.conf")

	err := os.WriteFile(configFile, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	return configFile
}

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
				t.Helper()
				return createTestConfig(t, "")
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
				t.Helper()
				return createTestConfig(t, "access-tokens = github.com=gho_testtoken123456789\n")
			},
			setupProviders: func() {
				// Clear registry and add mock provider
				provider.SetRegistry(make(map[string]*provider.Registration))
				setupMockGitHubProvider(true)
			},
			expectedOutput: []string{
				"Access Tokens (1 configured",
				"github.com",
				"Provider  github",
				"User      testuser (Test User)",
				"Token     gho_******89",
				"Scopes    repo, read:user",
				"Status    ✓ Valid",
			},
			expectError: false,
		},
		{
			name: "multiple tokens with one invalid",
			setupConfig: func(t *testing.T) string {
				t.Helper()
				return createTestConfig(t, "access-tokens = github.com=gho_validtoken123456 gitlab.com=glpat_invalidtoken789\n")
			},
			setupProviders: func() {
				provider.SetRegistry(make(map[string]*provider.Registration))
				// Override GitHub provider with different user info
				provider.RegisterProvider("github", provider.Registration{
					New: func(cfg provider.Config) provider.Provider {
						return &mockStatusProvider{
							name:     "github",
							host:     cfg.Host,
							valid:    true,
							scopes:   []string{"repo"},
							username: "ghuser",
							fullName: "GitHub User",
						}
					},
					Detect: func(_ context.Context, _ *http.Client, host string) (provider.Provider, error) {
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
				setupMockGitLabProvider(false)
			},
			expectedOutput: []string{
				"Access Tokens (2 configured",
				"github.com",
				"Provider  github",
				"User      ghuser (GitHub User)",
				"Token     gho_******56",
				"Status    ✓ Valid",
				"gitlab.com",
				"Provider  gitlab",
				"Token     glpa********",
				"Status    ✗ Invalid - 401 Unauthorized",
			},
			expectError: false,
		},
		{
			name: "unknown provider",
			setupConfig: func(t *testing.T) string {
				t.Helper()
				return createTestConfig(t, "access-tokens = unknown.host.com=token123456789012345\n")
			},
			setupProviders: func() {
				provider.SetRegistry(make(map[string]*provider.Registration))
				// No need to register unknown provider - it's handled internally
			},
			expectedOutput: []string{
				"unknown.host.com",
				"Provider  unknown",
				"Status    ⚠ Unknown (unverified)",
				"Token     toke********",
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
			output, err := captureStatusOutput(t)

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

// mockStatusProvider implements Provider interface for status command testing.
type mockStatusProvider struct {
	name       string
	host       string
	valid      bool
	validError error
	scopes     []string
	username   string
	fullName   string
}

func (m *mockStatusProvider) Name() string { return m.name }
func (m *mockStatusProvider) Host() string { return m.host }

func (m *mockStatusProvider) Authenticate(_ context.Context) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *mockStatusProvider) ValidateToken(_ context.Context, _ string) (provider.ValidationStatus, error) {
	if m.valid {
		return provider.ValidationStatusValid, nil
	}

	return provider.ValidationStatusInvalid, m.validError
}

func (m *mockStatusProvider) GetScopes() []string {
	return m.scopes
}

func (m *mockStatusProvider) GetTokenScopes(_ context.Context, _ string) ([]string, error) {
	if !m.valid {
		return nil, fmt.Errorf("invalid token")
	}

	return m.scopes, nil
}

func (m *mockStatusProvider) GetUserInfo(_ context.Context, _ string) (string, string, error) {
	if !m.valid {
		return "", "", fmt.Errorf("invalid token")
	}

	return m.username, m.fullName, nil
}

func TestStatusCommandIntegration(t *testing.T) {
	// Test that the status command is properly registered
	if statusCmd == nil {
		t.Error("statusCmd should not be nil")
	}

	if statusCmd.Use != "status [host...]" {
		t.Errorf("expected Use to be 'status [host...]', got %q", statusCmd.Use)
	}

	if statusCmd.RunE == nil {
		t.Error("statusCmd.RunE should not be nil")
	}
}
