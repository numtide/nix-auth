package provider

import (
	"context"
	"net/http"
)

func init() {
	RegisterProvider("forgejo", ProviderRegistration{
		New: func(cfg ProviderConfig) Provider {
			return &ForgejoProvider{
				PersonalAccessTokenProvider: PersonalAccessTokenProvider{
					providerName: "forgejo",
					defaultHost:  "", // No default host for Forgejo
					host:         cfg.Host,
				},
			}
		},
		Detect:      NewForgejoProviderForHost,
		DefaultHost: "", // No default host for Forgejo
	})

	// Codeberg is just an alias with a specific host
	RegisterProvider("codeberg", ProviderRegistration{
		New: func(cfg ProviderConfig) Provider {
			return &ForgejoProvider{
				PersonalAccessTokenProvider: PersonalAccessTokenProvider{
					providerName: "forgejo",
					defaultHost:  "codeberg.org",
					host:         cfg.Host,
				},
			}
		},
		// No detector for codeberg alias
		Detect:      nil,
		DefaultHost: "codeberg.org",
	})
}

// NewForgejoProviderForHost attempts to create a Forgejo provider for the given host
// Returns nil, nil if the host is not a Forgejo instance
// Returns nil, error if there was a network error during detection
func NewForgejoProviderForHost(ctx context.Context, client *http.Client, host string) (Provider, error) {
	provider, err := detectGiteaOrForgejo(ctx, client, host)
	if err != nil {
		return nil, err
	}

	// Check if it's actually a Forgejo provider
	if provider != nil && provider.Name() == "forgejo" {
		return provider, nil
	}

	return nil, nil // Not a Forgejo instance
}

type ForgejoProvider struct {
	PersonalAccessTokenProvider
}
