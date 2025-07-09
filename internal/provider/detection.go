package provider

import (
	"context"
	"net/http"
	"time"
)

// Detect attempts to identify the provider type by querying various API endpoints
func Detect(ctx context.Context, host, clientID string) (Provider, error) {
	// Create a client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try each registered provider in preferred order
	for _, name := range ListForDetection() {
		reg, ok := registry[name]
		if !ok || reg.Detect == nil {
			continue
		}

		provider, err := reg.Detect(ctx, client, host)
		if err != nil {
			// Network error - return unknown provider with the host set
			return NewUnknownProvider(host), nil
		}
		if provider != nil {
			// Found a matching provider
			// If clientID is provided, recreate with proper config
			if clientID != "" {
				cfg := ProviderConfig{
					Host:     host,
					ClientID: clientID,
				}
				return reg.New(cfg), nil
			}
			return provider, nil
		}
		// provider is nil - this detector doesn't match, try the next one
	}

	// If no specific provider matched, use the unknown provider
	return NewUnknownProvider(host), nil
}
