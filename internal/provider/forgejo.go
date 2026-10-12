package provider

import (
	"context"
	"net/http"
)

func init() {
	RegisterProvider(forgejoName, Registration{
		New: func(cfg Config) Provider {
			return &ForgejoProvider{
				PersonalAccessTokenProvider: PersonalAccessTokenProvider{
					providerName: forgejoName,
					defaultHost:  "", // No default host for Forgejo
					host:         cfg.Host,
				},
			}
		},
		Detect:      NewForgejoProviderForHost,
		DefaultHost: "", // No default host for Forgejo
	})

	// Codeberg is just an alias with a specific host
	RegisterProvider("codeberg", Registration{
		New: func(cfg Config) Provider {
			return &ForgejoProvider{
				PersonalAccessTokenProvider: PersonalAccessTokenProvider{
					providerName: forgejoName,
					defaultHost:  codebergHost,
					host:         cfg.Host,
				},
			}
		},
		// No detector for codeberg alias
		Detect:      nil,
		DefaultHost: codebergHost,
	})
}

// NewForgejoProviderForHost attempts to create a Forgejo provider for the given host
// Returns nil, nil if the host is not a Forgejo instance
// Returns nil, error if there was a network error during detection.
func NewForgejoProviderForHost(ctx context.Context, client *http.Client, host string) (Provider, error) {
	provider, err := detectGiteaOrForgejo(ctx, client, host)
	if err != nil {
		return nil, err
	}

	// Check if it's actually a Forgejo provider
	if provider != nil && provider.Name() == forgejoName {
		return provider, nil
	}

	return nil, nil // Not a Forgejo instance
}

// ForgejoProvider implements authentication for Forgejo instances.
type ForgejoProvider struct {
	PersonalAccessTokenProvider
}
