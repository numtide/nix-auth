package cmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/numtide/nix-auth/internal/provider"
	"github.com/spf13/cobra"
)

// mockSetTokenProvider implements provider.Provider for testing
type mockSetTokenProvider struct {
	name           string
	host           string
	validateResult provider.ValidationStatus
	validateError  error
}

func (m *mockSetTokenProvider) Name() string { return m.name }
func (m *mockSetTokenProvider) Host() string { return m.host }
func (m *mockSetTokenProvider) Authenticate(ctx context.Context) (string, error) {
	return "", nil
}
func (m *mockSetTokenProvider) ValidateToken(ctx context.Context, token string) (provider.ValidationStatus, error) {
	return m.validateResult, m.validateError
}
func (m *mockSetTokenProvider) GetUserInfo(ctx context.Context, token string) (string, string, error) {
	return "", "", nil
}
func (m *mockSetTokenProvider) GetScopes() []string {
	return []string{}
}
func (m *mockSetTokenProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	return []string{}, nil
}

func TestRunSetToken(t *testing.T) {
	// Save original values
	originalConfigPath := configPath
	originalRegistry := provider.GetRegistry()
	originalForce := setTokenForce
	originalProvider := setTokenProvider

	defer func() {
		configPath = originalConfigPath
		provider.SetRegistry(originalRegistry)
		setTokenForce = originalForce
		setTokenProvider = originalProvider
	}()

	tests := []struct {
		name            string
		args            []string
		setupFlags      func()
		setupConfig     func(t *testing.T) string
		setupProviders  func()
		mockStdin       string
		expectedOutputs []string
		expectError     bool
		errorContains   string
	}{
		{
			name: "set new token with arguments",
			args: []string{"test.example.com", "test-token-123"},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			expectedOutputs: []string{
				"Successfully set token for test.example.com: test****-123",
				"Config saved to:",
			},
		},
		{
			name: "set new token interactively",
			args: []string{"test.example.com"},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			mockStdin: "interactive-token-456\n",
			expectedOutputs: []string{
				"Enter token for test.example.com:",
				"Successfully set token for test.example.com: inte****-456",
			},
		},
		{
			name: "replace existing token with force flag",
			args: []string{"test.example.com", "new-token-789"},
			setupFlags: func() {
				setTokenForce = true
			},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := "access-tokens = test.example.com=old-token-123\n"
				if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			expectedOutputs: []string{
				"Successfully set token for test.example.com: new-****-789",
			},
		},
		{
			name: "cancel replacement of existing token",
			args: []string{"test.example.com", "new-token"},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := "access-tokens = test.example.com=old-token-123\n"
				if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			mockStdin: "n\n",
			expectedOutputs: []string{
				"Token already exists for test.example.com: old-****-123",
				"Replace it? (y/N):",
				"Operation cancelled",
			},
			expectError: false,
		},
		{
			name: "accept replacement of existing token",
			args: []string{"test.example.com", "new-token-789"},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := "access-tokens = test.example.com=old-token-123\n"
				if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			mockStdin: "y\n",
			expectedOutputs: []string{
				"Token already exists for test.example.com: old-****-123",
				"Replace it? (y/N):",
				"Successfully set token for test.example.com: new-****-789",
			},
		},
		{
			name: "validate token with specified provider",
			args: []string{"test.example.com", "valid-token"},
			setupFlags: func() {
				setTokenProvider = "test-provider"
			},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				reg := make(map[string]*provider.ProviderRegistration)
				reg["test-provider"] = &provider.ProviderRegistration{
					DefaultHost: "test.example.com",
					New: func(cfg provider.ProviderConfig) provider.Provider {
						return &mockSetTokenProvider{
							name:           "test-provider",
							host:           cfg.Host,
							validateResult: provider.ValidationStatusValid,
						}
					},
					Detect: func(ctx context.Context, client *http.Client, host string) (provider.Provider, error) {
						return nil, nil
					},
				}
				provider.SetRegistry(reg)
			},
			expectedOutputs: []string{
				"Validating token with test-provider provider...",
				"Token validated successfully",
				"Successfully set token for test.example.com: vali****oken",
			},
		},
		{
			name: "fail validation with specified provider",
			args: []string{"test.example.com", "invalid-token"},
			setupFlags: func() {
				setTokenProvider = "test-provider"
			},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				reg := make(map[string]*provider.ProviderRegistration)
				reg["test-provider"] = &provider.ProviderRegistration{
					DefaultHost: "test.example.com",
					New: func(cfg provider.ProviderConfig) provider.Provider {
						return &mockSetTokenProvider{
							name:           "test-provider",
							host:           cfg.Host,
							validateResult: provider.ValidationStatusInvalid,
						}
					},
					Detect: func(ctx context.Context, client *http.Client, host string) (provider.Provider, error) {
						return nil, nil
					},
				}
				provider.SetRegistry(reg)
			},
			expectedOutputs: []string{
				"Validating token with test-provider provider...",
			},
			expectError:   true,
			errorContains: "token is not valid",
		},
		{
			name: "unknown provider specified",
			args: []string{"test.example.com", "token"},
			setupFlags: func() {
				setTokenProvider = "unknown-provider"
			},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				provider.SetRegistry(make(map[string]*provider.ProviderRegistration))
			},
			expectError:   true,
			errorContains: "unknown provider: unknown-provider",
		},
		{
			name: "empty token from arguments",
			args: []string{"test.example.com", ""},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			expectError:   true,
			errorContains: "token cannot be empty",
		},
		{
			name: "empty token from stdin",
			args: []string{"test.example.com"},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			mockStdin: "   \n",
			expectedOutputs: []string{
				"Enter token for test.example.com:",
			},
			expectError:   true,
			errorContains: "token cannot be empty",
		},
		{
			name: "auto-detect provider and warn on validation failure",
			args: []string{"test.example.com", "maybe-valid-token"},
			setupConfig: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				reg := make(map[string]*provider.ProviderRegistration)
				reg["test-provider"] = &provider.ProviderRegistration{
					DefaultHost: "test.example.com",
					New: func(cfg provider.ProviderConfig) provider.Provider {
						return &mockSetTokenProvider{
							name:           "test-provider",
							host:           cfg.Host,
							validateResult: provider.ValidationStatusInvalid,
						}
					},
					Detect: func(ctx context.Context, client *http.Client, host string) (provider.Provider, error) {
						if host == "test.example.com" {
							return &mockSetTokenProvider{
								name:           "test-provider",
								host:           host,
								validateResult: provider.ValidationStatusInvalid,
							}, nil
						}
						return nil, nil
					},
				}
				provider.SetRegistry(reg)
			},
			expectedOutputs: []string{
				"Detected test-provider provider, validating token...",
				"Warning: token may not be valid",
				"Successfully set token for test.example.com: mayb****oken",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			setTokenForce = false
			setTokenProvider = ""

			// Setup flags if provided
			if tt.setupFlags != nil {
				tt.setupFlags()
			}

			// Setup config
			configPath = tt.setupConfig(t)

			// Setup providers
			if tt.setupProviders != nil {
				tt.setupProviders()
			} else {
				provider.SetRegistry(make(map[string]*provider.ProviderRegistration))
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Mock stdin if needed
			var oldStdin *os.File
			if tt.mockStdin != "" {
				oldStdin = os.Stdin
				stdinR, stdinW, _ := os.Pipe()
				os.Stdin = stdinR
				go func() {
					defer stdinW.Close()
					io.WriteString(stdinW, tt.mockStdin)
				}()
			}

			// Create command and run
			cmd := &cobra.Command{}
			err := setTokenCmd.RunE(cmd, tt.args)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			io.Copy(&buf, r)

			// Restore stdin
			if oldStdin != nil {
				os.Stdin = oldStdin
			}

			output := buf.String()

			// Check error
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q but got %q", tt.errorContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			for _, expected := range tt.expectedOutputs {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q but got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestSetTokenCommandFlags(t *testing.T) {
	// Test that flags are properly defined
	if setTokenCmd.Flags().Lookup("force") == nil {
		t.Error("Expected 'force' flag to be defined")
	}

	if setTokenCmd.Flags().Lookup("provider") == nil {
		t.Error("Expected 'provider' flag to be defined")
	}

	// Test flag shortcuts
	forceFlag := setTokenCmd.Flags().ShorthandLookup("f")
	if forceFlag == nil || forceFlag.Name != "force" {
		t.Error("Expected 'f' to be shorthand for 'force'")
	}

	providerFlag := setTokenCmd.Flags().ShorthandLookup("p")
	if providerFlag == nil || providerFlag.Name != "provider" {
		t.Error("Expected 'p' to be shorthand for 'provider'")
	}
}

func TestSetTokenCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "no arguments",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "one argument (host only)",
			args:        []string{"test.example.com"},
			expectError: false,
		},
		{
			name:        "two arguments (host and token)",
			args:        []string{"test.example.com", "token"},
			expectError: false,
		},
		{
			name:        "three arguments (too many)",
			args:        []string{"test.example.com", "token", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setTokenCmd.Args(nil, tt.args)
			if tt.expectError && err == nil {
				t.Error("Expected error for args validation but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for args validation: %v", err)
			}
		})
	}
}