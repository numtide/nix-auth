package provider

import (
	"context"
	"net/http"
)

// ValidationStatus represents the result of token validation
type ValidationStatus int

const (
	// ValidationStatusValid indicates the token is valid
	ValidationStatusValid ValidationStatus = iota
	// ValidationStatusInvalid indicates the token is invalid
	ValidationStatusInvalid
	// ValidationStatusUnknown indicates the token cannot be verified
	ValidationStatusUnknown
)

// Provider defines the interface for authentication providers
type Provider interface {
	// Name returns the provider name (e.g., "github", "gitlab")
	Name() string

	// Host returns the host for this provider
	Host() string

	// Authenticate performs the OAuth flow and returns an access token
	Authenticate(ctx context.Context) (string, error)

	// ValidateToken checks if a token is valid
	ValidateToken(ctx context.Context, token string) (ValidationStatus, error)

	// GetUserInfo retrieves the authenticated user's information
	// Returns username and full name (full name may be empty)
	GetUserInfo(ctx context.Context, token string) (username, fullName string, err error)

	// GetScopes returns the required scopes for this provider
	GetScopes() []string

	// GetTokenScopes returns the actual scopes of a token
	GetTokenScopes(ctx context.Context, token string) ([]string, error)
}

// ProviderConfig contains configuration for creating a provider
type ProviderConfig struct {
	Host     string
	ClientID string
}

// NewProviderFunc is a function that creates a new provider instance with configuration
type NewProviderFunc func(cfg ProviderConfig) Provider

// DetectFunc is a function that attempts to create a provider for a given host
// Returns nil, nil if the host is not supported by this provider
// Returns nil, error if there was a network error during detection
type DetectFunc func(ctx context.Context, client *http.Client, host string) (Provider, error)

// ProviderRegistration contains constructor, detector, and default host for a provider
type ProviderRegistration struct {
	New         NewProviderFunc
	Detect      DetectFunc
	DefaultHost string // Default host for this provider (e.g., "github.com" for GitHub)
}

// registry holds provider registrations
var registry = make(map[string]*ProviderRegistration)

// GetRegistry returns the registry (for testing)
func GetRegistry() map[string]*ProviderRegistration {
	return registry
}

// SetRegistry sets the registry (for testing)
func SetRegistry(r map[string]*ProviderRegistration) {
	registry = r
}

// RegisterProvider registers both factory and detector for a provider
func RegisterProvider(name string, reg ProviderRegistration) {
	registry[name] = &reg
}

// GetRegistration returns the provider registration by name
func GetRegistration(name string) (*ProviderRegistration, bool) {
	reg, ok := registry[name]
	return reg, ok
}

// Get creates a new instance of a provider by name with default configuration
func Get(name string) (Provider, bool) {
	reg, ok := registry[name]
	if !ok {
		return nil, false
	}
	// Use default host from registration
	cfg := ProviderConfig{
		Host: reg.DefaultHost,
	}
	return reg.New(cfg), true
}

// GetWithConfig creates a new instance of a provider by name with custom configuration
func GetWithConfig(name string, cfg ProviderConfig) (Provider, bool) {
	reg, ok := registry[name]
	if !ok {
		return nil, false
	}
	// If no host is provided, use the default
	if cfg.Host == "" {
		cfg.Host = reg.DefaultHost
	}
	return reg.New(cfg), true
}

// List returns all registered provider names
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// ListForDetection returns provider names in the order they should be tried for detection
func ListForDetection() []string {
	// Define preferred order for detection
	// GitHub and GitLab are tried first as they're most common
	preferredOrder := []string{"github", "gitlab", "gitea", "forgejo"}

	result := []string{}
	// Add providers in preferred order if they exist
	for _, name := range preferredOrder {
		if _, exists := registry[name]; exists {
			result = append(result, name)
		}
	}

	// Add any other registered providers not in the preferred list
	for name := range registry {
		found := false
		for _, preferred := range preferredOrder {
			if name == preferred {
				found = true
				break
			}
		}
		if !found && name != "codeberg" { // Skip codeberg as it's an alias
			result = append(result, name)
		}
	}

	return result
}
