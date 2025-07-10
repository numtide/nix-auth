// Package provider implements authentication providers for various Git hosting services.
package provider

import (
	"context"
	"net/http"
	"time"
)

const (
	// detectionTimeout is the timeout for provider detection requests.
	detectionTimeout = 3 * time.Second
)

// Detect attempts to identify the provider type by querying various API endpoints.
func Detect(ctx context.Context, host, clientID string) (Provider, error) {
	// Create a client with timeout
	client := &http.Client{
		Timeout: detectionTimeout,
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
			return NewUnknownProvider(host), nil //nolint:nilerr // Network errors during detection are not fatal
		}

		if provider != nil {
			// Found a matching provider
			// If clientID is provided, recreate with proper config
			if clientID != "" {
				cfg := Config{
					Host:     host,
					ClientID: clientID,
				}

				return reg.New(cfg), nil
			}

			return provider, nil
		}
	}

	// If no specific provider matched, use the unknown provider
	return NewUnknownProvider(host), nil
}
