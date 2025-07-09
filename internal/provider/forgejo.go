package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func init() {
	Register("forgejo", &ForgejoProvider{
		PersonalAccessTokenProvider: PersonalAccessTokenProvider{
			providerName: "forgejo",
			defaultHost:  "", // No default host for Forgejo
		},
	})
	Register("codeberg", &ForgejoProvider{
		PersonalAccessTokenProvider: PersonalAccessTokenProvider{
			providerName: "forgejo",
			defaultHost:  "codeberg.org",
			host:         "codeberg.org",
		},
	})
}

type ForgejoProvider struct {
	PersonalAccessTokenProvider
}

// DetectHost checks if the given host is a Forgejo instance
func (f *ForgejoProvider) DetectHost(ctx context.Context, client *http.Client, host string) bool {
	// Known Forgejo/Codeberg host
	if strings.ToLower(host) == "codeberg.org" {
		return true
	}

	// For other hosts, check if it's a Forgejo instance using the version endpoint
	baseURL := fmt.Sprintf("https://%s", host)
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/version", baseURL), nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var data struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return false
		}
		// Forgejo includes "forgejo" in the version string
		if strings.Contains(strings.ToLower(data.Version), "forgejo") {
			return true
		}
	}
	return false
}
