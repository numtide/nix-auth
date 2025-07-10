package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// detectGiteaOrForgejo detects whether a host is running Gitea or Forgejo.
func detectGiteaOrForgejo(ctx context.Context, client *http.Client, host string) (Provider, error) {
	// Known hosts
	lowerHost := strings.ToLower(host)

	// Codeberg is known to be Forgejo
	if lowerHost == "codeberg.org" {
		return &ForgejoProvider{
			PersonalAccessTokenProvider: PersonalAccessTokenProvider{
				providerName: "forgejo",
				defaultHost:  "codeberg.org",
				host:         host,
			},
		}, nil
	}

	// Gitea.com and gitea.io are known Gitea instances
	if lowerHost == "gitea.com" || lowerHost == "gitea.io" {
		return &GiteaProvider{
			PersonalAccessTokenProvider: PersonalAccessTokenProvider{
				providerName: "gitea",
				defaultHost:  "gitea.com",
				host:         host,
			},
		}, nil
	}

	// For other hosts, check the version endpoint
	baseURL := fmt.Sprintf("https://%s", host)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/version", baseURL), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // cleanup

	if resp.StatusCode == http.StatusOK {
		var data struct {
			Version string `json:"version"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, nil // Not a Gitea/Forgejo instance
		}

		// Check if it's Forgejo (includes "forgejo" in version string)
		if strings.Contains(strings.ToLower(data.Version), "forgejo") {
			return &ForgejoProvider{
				PersonalAccessTokenProvider: PersonalAccessTokenProvider{
					providerName: "forgejo",
					defaultHost:  "",
					host:         host,
				},
			}, nil
		}

		// Otherwise it's Gitea
		if data.Version != "" {
			return &GiteaProvider{
				PersonalAccessTokenProvider: PersonalAccessTokenProvider{
					providerName: "gitea",
					defaultHost:  "gitea.com",
					host:         host,
				},
			}, nil
		}
	}

	return nil, nil // Not a Gitea/Forgejo instance
}
