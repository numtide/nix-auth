package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetect_Integration(t *testing.T) {
	// Save original registry
	originalRegistry := registry
	defer func() {
		registry = originalRegistry
	}()

	// Create test providers with mock servers
	tests := []struct {
		name             string
		setupServer      func() *httptest.Server
		expectedProvider string
		expectError      bool
	}{
		{
			name: "no matching API returns unknown",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedProvider: "unknown",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			// Extract host from server URL
			host := strings.TrimPrefix(server.URL, "http://")

			// No need to setup registry anymore - detection uses hardcoded providers

			// Test detection
			ctx := context.Background()
			provider, err := Detect(ctx, host, "")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if provider == nil {
					t.Errorf("expected provider but got nil")
				} else if provider.Name() != tt.expectedProvider {
					t.Errorf("expected provider %q, got %q", tt.expectedProvider, provider.Name())
				}
				// Verify host was set
				if provider != nil && provider.Host() != host {
					t.Errorf("expected host %q, got %q", host, provider.Host())
				}
			}
		})
	}
}

// testDetectionProvider is a minimal provider for testing detection
type testDetectionProvider struct {
	name string
	host string
}

func (t *testDetectionProvider) Name() string { return t.name }
func (t *testDetectionProvider) Host() string { return t.host }
func (t *testDetectionProvider) Authenticate(ctx context.Context) (string, error) {
	return "", nil
}
func (t *testDetectionProvider) ValidateToken(ctx context.Context, token string) (ValidationStatus, error) {
	return ValidationStatusValid, nil
}
func (t *testDetectionProvider) GetUserInfo(ctx context.Context, token string) (string, string, error) {
	return "", "", nil
}
func (t *testDetectionProvider) GetScopes() []string {
	return nil
}
func (t *testDetectionProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	return nil, nil
}
