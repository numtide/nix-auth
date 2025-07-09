package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestRegistry(t *testing.T) {
	// Save original registry
	originalRegistry := Registry
	defer func() {
		Registry = originalRegistry
	}()

	// Test with empty registry
	Registry = make(map[string]Provider)

	// Test Get with non-existent provider
	_, exists := Get("nonexistent")
	if exists {
		t.Errorf("expected Get to return false for non-existent provider")
	}

	// Test Register
	mockProvider := &mockProvider{name: "mock", host: "mock.example.com"}
	Register("mock", mockProvider)

	// Test Get with existing provider
	p, exists := Get("mock")
	if !exists {
		t.Errorf("expected Get to return true for registered provider")
	}
	if p != mockProvider {
		t.Errorf("expected Get to return the registered provider")
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

// mockProvider implements Provider interface for testing
type mockProvider struct {
	name     string
	host     string
	clientID string
}

func (m *mockProvider) Name() string                { return m.name }
func (m *mockProvider) Host() string                { return m.host }
func (m *mockProvider) SetHost(host string)         { m.host = host }
func (m *mockProvider) SetClientID(clientID string) { m.clientID = clientID }
func (m *mockProvider) DetectHost(ctx context.Context, client *http.Client, host string) bool {
	return host == m.host
}
func (m *mockProvider) Authenticate(ctx context.Context) (string, error) {
	return "mock-token", nil
}
func (m *mockProvider) ValidateToken(ctx context.Context, token string) error {
	if token == "invalid" {
		return fmt.Errorf("invalid token")
	}
	return nil
}
func (m *mockProvider) GetUserInfo(ctx context.Context, token string) (string, string, error) {
	return "mockuser", "Mock User", nil
}
func (m *mockProvider) GetScopes() []string {
	return []string{"read", "write"}
}
func (m *mockProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	return []string{"read", "write"}, nil
}

func TestDetectProviderFromHost(t *testing.T) {
	// Save original registry
	originalRegistry := Registry
	defer func() {
		Registry = originalRegistry
	}()

	tests := []struct {
		name             string
		host             string
		setupProviders   func()
		expectedProvider string
		expectError      bool
	}{
		{
			name: "detect registered provider",
			host: "test.example.com",
			setupProviders: func() {
				Registry = make(map[string]Provider)
				mockP := &mockProvider{name: "test", host: "test.example.com"}
				Register("test", mockP)
			},
			expectedProvider: "test",
			expectError:      false,
		},
		{
			name: "no matching provider",
			host: "unknown.example.com",
			setupProviders: func() {
				Registry = make(map[string]Provider)
				mockP := &mockProvider{name: "test", host: "test.example.com"}
				Register("test", mockP)
			},
			expectedProvider: "",
			expectError:      true,
		},
		{
			name: "empty registry",
			host: "any.example.com",
			setupProviders: func() {
				Registry = make(map[string]Provider)
			},
			expectedProvider: "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupProviders()

			ctx := context.Background()
			provider, err := DetectProviderFromHost(ctx, tt.host)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if provider != tt.expectedProvider {
					t.Errorf("expected provider %q, got %q", tt.expectedProvider, provider)
				}
			}
		})
	}
}

