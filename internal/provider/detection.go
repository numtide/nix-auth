package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// DetectProviderFromHost attempts to identify the provider type by querying various API endpoints
func DetectProviderFromHost(ctx context.Context, host string) (string, error) {
	// Create a client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try each registered provider
	for name, provider := range Registry {
		// Create a new instance to avoid mutating the registered provider
		p := provider
		if p.DetectHost(ctx, client, host) {
			return name, nil
		}
	}

	return "", fmt.Errorf("unable to detect provider type for host: %s", host)
}