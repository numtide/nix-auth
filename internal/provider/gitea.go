package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func init() {
	Register("gitea", &GiteaProvider{
		PersonalAccessTokenProvider: PersonalAccessTokenProvider{
			providerName: "gitea",
			defaultHost:  "gitea.com",
		},
	})
}

type GiteaProvider struct {
	PersonalAccessTokenProvider
}

// DetectHost checks if the given host is a Gitea instance
func (g *GiteaProvider) DetectHost(ctx context.Context, client *http.Client, host string) bool {
	// Known Gitea hosts
	lowerHost := strings.ToLower(host)
	if lowerHost == "gitea.com" || lowerHost == "gitea.io" {
		return true
	}

	// For other hosts, check if it's a Gitea instance using the version endpoint
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
		// Check if it's NOT Forgejo (Forgejo includes "forgejo" in version string)
		if data.Version != "" && !strings.Contains(strings.ToLower(data.Version), "forgejo") {
			return true
		}
	}
	return false
}
