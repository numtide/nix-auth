package provider

import (
	"context"
	"net/http"
)

func init() {
	RegisterProvider("gitea", Registration{
		New: func(cfg Config) Provider {
			return &GiteaProvider{
				PersonalAccessTokenProvider: PersonalAccessTokenProvider{
					providerName: "gitea",
					defaultHost:  "gitea.com",
					host:         cfg.Host,
				},
			}
		},
		Detect:      NewGiteaProviderForHost,
		DefaultHost: "gitea.com",
	})
}

// NewGiteaProviderForHost attempts to create a Gitea provider for the given host
// Returns nil, nil if the host is not a Gitea instance
// Returns nil, error if there was a network error during detection
func NewGiteaProviderForHost(ctx context.Context, client *http.Client, host string) (Provider, error) {
	provider, err := detectGiteaOrForgejo(ctx, client, host)
	if err != nil {
		return nil, err
	}

	// Check if it's actually a Gitea provider
	if provider != nil && provider.Name() == "gitea" {
		return provider, nil
	}

	return nil, nil // Not a Gitea instance
}

type GiteaProvider struct {
	PersonalAccessTokenProvider
}
