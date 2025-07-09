package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetectProviderFromHost_Integration(t *testing.T) {
	// Save original registry
	originalRegistry := Registry
	defer func() {
		Registry = originalRegistry
	}()

	// Create test providers with mock servers
	tests := []struct {
		name             string
		setupServer      func() *httptest.Server
		expectedProvider string
		expectError      bool
	}{
		{
			name: "github-like API",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/v3" {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"current_user_url":"https://api.github.com/user"}`))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectedProvider: "github-test",
			expectError:      false,
		},
		{
			name: "gitea-like API",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/v1/version" {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"version":"1.21.0"}`))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectedProvider: "gitea-test",
			expectError:      false,
		},
		{
			name: "no matching API",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedProvider: "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			// Extract host from server URL
			host := strings.TrimPrefix(server.URL, "http://")

			// Setup registry with test providers
			Registry = make(map[string]Provider)

			// GitHub-like provider
			Register("github-test", &testDetectionProvider{
				name: "github-test",
				detectFunc: func(ctx context.Context, client *http.Client, h string) bool {
					resp, err := client.Get("http://" + h + "/api/v3")
					if err != nil {
						return false
					}
					defer resp.Body.Close()
					return resp.StatusCode == http.StatusOK
				},
			})

			// Gitea-like provider
			Register("gitea-test", &testDetectionProvider{
				name: "gitea-test",
				detectFunc: func(ctx context.Context, client *http.Client, h string) bool {
					resp, err := client.Get("http://" + h + "/api/v1/version")
					if err != nil {
						return false
					}
					defer resp.Body.Close()
					return resp.StatusCode == http.StatusOK
				},
			})

			// Test detection
			ctx := context.Background()
			provider, err := DetectProviderFromHost(ctx, host)

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
	name       string
	host       string
	detectFunc func(context.Context, *http.Client, string) bool
}

func (t *testDetectionProvider) Name() string                { return t.name }
func (t *testDetectionProvider) Host() string                { return t.host }
func (t *testDetectionProvider) SetHost(host string)         { t.host = host }
func (t *testDetectionProvider) SetClientID(clientID string) {}
func (t *testDetectionProvider) DetectHost(ctx context.Context, client *http.Client, host string) bool {
	if t.detectFunc != nil {
		return t.detectFunc(ctx, client, host)
	}
	return false
}
func (t *testDetectionProvider) Authenticate(ctx context.Context) (string, error) {
	return "", nil
}
func (t *testDetectionProvider) ValidateToken(ctx context.Context, token string) error {
	return nil
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
