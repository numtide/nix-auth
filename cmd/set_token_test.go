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

const (
	testExistingTokenConfig = "access-tokens = test.example.com=old-token-123\n"
)

// mockSetTokenProvider implements provider.Provider for testing.
type mockSetTokenProvider struct {
	name           string
	host           string
	validateResult provider.ValidationStatus
	validateError  error
}

func (m *mockSetTokenProvider) Name() string { return m.name }
func (m *mockSetTokenProvider) Host() string { return m.host }
func (m *mockSetTokenProvider) Authenticate(_ context.Context) (string, error) {
	return "", nil
}
func (m *mockSetTokenProvider) ValidateToken(_ context.Context, _ string) (provider.ValidationStatus, error) {
	return m.validateResult, m.validateError
}
func (m *mockSetTokenProvider) GetUserInfo(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}
func (m *mockSetTokenProvider) GetScopes() []string {
	return []string{}
}
func (m *mockSetTokenProvider) GetTokenScopes(_ context.Context, _ string) ([]string, error) {
	return []string{}, nil
}

// setupSetTokenTest saves and restores global state for tests.
func setupSetTokenTest(t *testing.T) {
	t.Helper()

	originalConfigPath := configPath
	originalRegistry := provider.GetRegistry()
	originalForce := setTokenForce
	originalProvider := setTokenProvider

	t.Cleanup(func() {
		configPath = originalConfigPath

		provider.SetRegistry(originalRegistry)

		setTokenForce = originalForce
		setTokenProvider = originalProvider
	})
}

// runSetTokenTest is a helper to run set-token command tests.
func runSetTokenTest(t *testing.T, tc struct {
	name            string
	args            []string
	setupFlags      func()
	setupConfig     func(t *testing.T) string
	setupProviders  func()
	mockStdin       string
	expectedOutputs []string
	expectError     bool
	errorContains   string
}) {
	t.Helper()

	// Reset flags
	setTokenForce = false
	setTokenProvider = ""

	// Setup flags if provided
	if tc.setupFlags != nil {
		tc.setupFlags()
	}

	// Setup config
	configPath = tc.setupConfig(t)

	// Setup providers
	if tc.setupProviders != nil {
		tc.setupProviders()
	} else {
		provider.SetRegistry(make(map[string]*provider.Registration))
	}

	// Capture output
	var buf bytes.Buffer

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Mock stdin if needed
	var oldStdin *os.File
	if tc.mockStdin != "" {
		oldStdin = os.Stdin
		stdinR, stdinW, _ := os.Pipe()
		os.Stdin = stdinR

		go func() {
			defer stdinW.Close() //nolint:errcheck // cleanup in test goroutine
			_, _ = io.WriteString(stdinW, tc.mockStdin)
		}()
	}

	// Create command and run
	cmd := &cobra.Command{}
	err := setTokenCmd.RunE(cmd, tc.args)

	// Restore stdout
	_ = w.Close()

	os.Stdout = oldStdout
	_, _ = io.Copy(&buf, r)

	// Restore stdin
	if oldStdin != nil {
		os.Stdin = oldStdin
	}

	output := buf.String()

	// Check error
	if tc.expectError {
		if err == nil {
			t.Errorf("Expected error but got none")
		} else if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
			t.Errorf("Expected error containing %q but got %q", tc.errorContains, err.Error())
		}
	} else if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check output
	for _, expected := range tc.expectedOutputs {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q but got:\n%s", expected, output)
		}
	}
}

func TestSetTokenBasicOperations(t *testing.T) {
	setupSetTokenTest(t)

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
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			expectedOutputs: []string{
				"Successfully set token for test.example.com: test********",
				"Config saved to:",
			},
		},
		{
			name: "set new token interactively",
			args: []string{"test.example.com"},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			mockStdin: "interactive-token-456\n",
			expectedOutputs: []string{
				"Enter token for test.example.com:",
				"Successfully set token for test.example.com: inte********",
			},
		},
		{
			name: "replace existing token with force flag",
			args: []string{"test.example.com", "new-token-789"},
			setupFlags: func() {
				setTokenForce = true
			},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := testExistingTokenConfig
				if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			expectedOutputs: []string{
				"Successfully set token for test.example.com: ********",
			},
		},
		{
			name: "cancel replacement of existing token",
			args: []string{"test.example.com", "new-token"},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := testExistingTokenConfig
				if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			mockStdin: "n\n",
			expectedOutputs: []string{
				"Token already exists for test.example.com: ********",
				"Replace it? (y/N):",
				"Operation cancelled",
			},
			expectError: false,
		},
		{
			name: "accept replacement of existing token",
			args: []string{"test.example.com", "new-token-789"},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				content := testExistingTokenConfig
				if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			mockStdin: "y\n",
			expectedOutputs: []string{
				"Token already exists for test.example.com: ********",
				"Replace it? (y/N):",
				"Successfully set token for test.example.com: ********",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSetTokenTest(t, tt)
		})
	}
}

func TestSetTokenProviderValidation(t *testing.T) {
	setupSetTokenTest(t)

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
			name: "validate token with specified provider",
			args: []string{"test.example.com", "valid-token"},
			setupFlags: func() {
				setTokenProvider = "test-provider"
			},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				reg := make(map[string]*provider.Registration)
				reg["test-provider"] = &provider.Registration{
					DefaultHost: "test.example.com",
					New: func(cfg provider.Config) provider.Provider {
						return &mockSetTokenProvider{
							name:           "test-provider",
							host:           cfg.Host,
							validateResult: provider.ValidationStatusValid,
						}
					},
					Detect: func(_ context.Context, _ *http.Client, _ string) (provider.Provider, error) {
						return nil, nil
					},
				}
				provider.SetRegistry(reg)
			},
			expectedOutputs: []string{
				"Validating token with test-provider provider...",
				"Token validated successfully",
				"Successfully set token for test.example.com: ********",
			},
		},
		{
			name: "fail validation with specified provider",
			args: []string{"test.example.com", "invalid-token"},
			setupFlags: func() {
				setTokenProvider = "test-provider"
			},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				reg := make(map[string]*provider.Registration)
				reg["test-provider"] = &provider.Registration{
					DefaultHost: "test.example.com",
					New: func(cfg provider.Config) provider.Provider {
						return &mockSetTokenProvider{
							name:           "test-provider",
							host:           cfg.Host,
							validateResult: provider.ValidationStatusInvalid,
						}
					},
					Detect: func(_ context.Context, _ *http.Client, _ string) (provider.Provider, error) {
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
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				provider.SetRegistry(make(map[string]*provider.Registration))
			},
			expectError:   true,
			errorContains: "unknown provider: unknown-provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSetTokenTest(t, tt)
		})
	}
}

func TestSetTokenErrorCases(t *testing.T) {
	setupSetTokenTest(t)

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
			name: "empty token from arguments",
			args: []string{"test.example.com", ""},
			setupConfig: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
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
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
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
				t.Helper()
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "nix.conf")
				if err := os.WriteFile(configFile, []byte(""), 0600); err != nil {
					t.Fatal(err)
				}
				return configFile
			},
			setupProviders: func() {
				reg := make(map[string]*provider.Registration)
				reg["test-provider"] = &provider.Registration{
					DefaultHost: "test.example.com",
					New: func(cfg provider.Config) provider.Provider {
						return &mockSetTokenProvider{
							name:           "test-provider",
							host:           cfg.Host,
							validateResult: provider.ValidationStatusInvalid,
						}
					},
					Detect: func(_ context.Context, _ *http.Client, host string) (provider.Provider, error) {
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
				"Successfully set token for test.example.com: mayb********",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSetTokenTest(t, tt)
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
