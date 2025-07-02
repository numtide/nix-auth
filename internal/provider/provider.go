package provider

import "context"

// Provider defines the interface for authentication providers
type Provider interface {
	// Name returns the provider name (e.g., "github", "gitlab")
	Name() string

	// Host returns the default host for this provider
	Host() string

	// SetHost sets a custom host for this provider
	SetHost(host string)

	// SetClientID sets a custom OAuth client ID for this provider
	SetClientID(clientID string)

	// Authenticate performs the OAuth flow and returns an access token
	Authenticate(ctx context.Context) (string, error)

	// ValidateToken checks if a token is valid
	ValidateToken(ctx context.Context, token string) error

	// GetUserInfo retrieves the authenticated user's information
	// Returns username and full name (full name may be empty)
	GetUserInfo(ctx context.Context, token string) (username, fullName string, err error)

	// GetScopes returns the required scopes for this provider
	GetScopes() []string

	// GetTokenScopes returns the actual scopes of a token
	GetTokenScopes(ctx context.Context, token string) ([]string, error)
}

// Registry holds all available providers
var Registry = make(map[string]Provider)

// Register adds a provider to the registry
func Register(name string, provider Provider) {
	Registry[name] = provider
}

// Get returns a provider by name
func Get(name string) (Provider, bool) {
	p, ok := Registry[name]
	return p, ok
}

// List returns all registered provider names
func List() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}
