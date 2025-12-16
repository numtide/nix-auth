package provider

import (
	"context"
	"fmt"
	"testing"
)

func TestRegistry(t *testing.T) {
	// Save original registry
	originalRegistry := registry
	defer func() {
		registry = originalRegistry
	}()

	// Test with empty registry
	registry = make(map[string]*Registration)

	// Test Get with non-existent provider
	_, exists := Get("nonexistent")
	if exists {
		t.Errorf("expected Get to return false for non-existent provider")
	}

	// Test RegisterProvider
	RegisterProvider("mock", Registration{
		New: func(cfg Config) Provider {
			return &mockProvider{
				name:     "mock",
				host:     cfg.Host,
				clientID: cfg.ClientID,
			}
		},
		Detect:      nil,
		DefaultHost: "mock.example.com",
	})

	// Test Get with existing provider
	p, exists := Get("mock")
	if !exists {
		t.Errorf("expected Get to return true for registered provider")
	}
	// Verify the provider has correct name and default host
	if p.Name() != "mock" {
		t.Errorf("expected provider name to be 'mock', got %q", p.Name())
	}

	if p.Host() != "mock.example.com" {
		t.Errorf("expected provider host to be 'mock.example.com', got %q", p.Host())
	}

	// Test List
	providers := List()
	if len(providers) != 1 {
		t.Errorf("expected List to return 1 provider, got %d", len(providers))
	}

	// Check that mock is in the list
	found := false

	for _, name := range providers {
		if name == "mock" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected List to contain 'mock', got %v", providers)
	}
}

// mockProvider implements Provider interface for testing.
type mockProvider struct {
	name     string
	host     string
	clientID string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Host() string { return m.host }
func (m *mockProvider) Authenticate(_ context.Context) (string, error) {
	return "mock-token", nil
}

func (m *mockProvider) ValidateToken(_ context.Context, token string) (ValidationStatus, error) {
	if token == "invalid" {
		return ValidationStatusInvalid, fmt.Errorf("invalid token")
	}

	return ValidationStatusValid, nil
}

func (m *mockProvider) GetUserInfo(_ context.Context, _ string) (string, string, error) {
	return "mockuser", "Mock User", nil
}

func (m *mockProvider) GetScopes() []string {
	return []string{"read", "write"}
}

func (m *mockProvider) GetTokenScopes(_ context.Context, _ string) ([]string, error) {
	return []string{"read", "write"}, nil
}

func TestDetect(t *testing.T) {
	// Save original registry
	originalRegistry := registry
	defer func() {
		registry = originalRegistry
	}()

	tests := []struct {
		name             string
		host             string
		setupProviders   func()
		expectedProvider string
		expectError      bool
	}{
		{
			name: "detect github.com",
			host: "github.com",
			setupProviders: func() {
				// No setup needed - detection uses hardcoded providers
			},
			expectedProvider: "github",
			expectError:      false,
		},
		{
			name: "no matching provider returns unknown",
			host: "unknown.example.com",
			setupProviders: func() {
				registry = make(map[string]*Registration)
				RegisterProvider("test", Registration{
					New: func(cfg Config) Provider {
						return &mockProvider{
							name: "test",
							host: cfg.Host,
						}
					},
					Detect:      nil,
					DefaultHost: "test.example.com",
				})
			},
			expectedProvider: "unknown",
			expectError:      false,
		},
		{
			name: "empty registry returns unknown",
			host: "any.example.com",
			setupProviders: func() {
				registry = make(map[string]*Registration)
			},
			expectedProvider: "unknown",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupProviders()

			ctx := context.Background()
			provider, err := Detect(ctx, tt.host, "")

			// Validate error expectation
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// If we expected an error, verify provider is nil
			if tt.expectError {
				if provider != nil {
					t.Errorf("expected nil provider on error, got %v", provider)
				}

				return
			}

			// Validate provider for success cases
			if provider == nil {
				t.Errorf("expected provider but got nil")
				return
			}

			if provider.Name() != tt.expectedProvider {
				t.Errorf("expected provider %q, got %q", tt.expectedProvider, provider.Name())
			}

			// Verify host was set
			if provider.Host() != tt.host {
				t.Errorf("expected host %q, got %q", tt.host, provider.Host())
			}
		})
	}
}
