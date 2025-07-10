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
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

			// Validate error expectation
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// If we expected an error and got one, we're done
			if tt.expectError {
				return
			}

			// Validate provider
			if provider == nil {
				t.Errorf("expected provider but got nil")
				return
			}

			if provider.Name() != tt.expectedProvider {
				t.Errorf("expected provider %q, got %q", tt.expectedProvider, provider.Name())
			}

			// Verify host was set
			if provider.Host() != host {
				t.Errorf("expected host %q, got %q", host, provider.Host())
			}
		})
	}
}
