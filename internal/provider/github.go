package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cli/oauth/device"
)

func init() {
	Register("github", &GitHubProvider{})
}

type GitHubProvider struct{}

// makeGitHubAPIRequest is a helper function to make authenticated requests to GitHub API
func (g *GitHubProvider) makeGitHubAPIRequest(ctx context.Context, token string, endpoint string) (*http.Response, error) {
	headers := map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	return makeAuthenticatedRequest(ctx, "GET", endpoint, token, "token "+token, headers)
}

func (g *GitHubProvider) Name() string {
	return "github"
}

func (g *GitHubProvider) Host() string {
	return "github.com"
}

func (g *GitHubProvider) GetScopes() []string {
	// Minimal scope needed for private repo access
	return []string{"repo"}
}

func (g *GitHubProvider) Authenticate(ctx context.Context) (string, error) {
	clientID := "178c6fc778ccc68e1d6a" // GitHub CLI's client ID - widely used for CLI tools
	scopes := g.GetScopes()

	httpClient := &http.Client{}

	// Request device code
	code, err := device.RequestCode(httpClient, "https://github.com/login/device/code", clientID, scopes)
	if err != nil {
		return "", fmt.Errorf("failed to request device code: %w", err)
	}

	DisplayDeviceCode(code.UserCode)
	DisplayURLAndOpenBrowser(code.VerificationURI)
	ShowWaitingMessage()

	// Wait for user to authorize
	accessToken, err := device.Wait(ctx, httpClient, "https://github.com/login/oauth/access_token", device.WaitOptions{
		ClientID:   clientID,
		DeviceCode: code,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return accessToken.Token, nil
}

func (g *GitHubProvider) ValidateToken(ctx context.Context, token string) error {
	resp, err := g.makeGitHubAPIRequest(ctx, token, "https://api.github.com/user")
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (g *GitHubProvider) GetUserInfo(ctx context.Context, token string) (username, fullName string, err error) {
	resp, err := g.makeGitHubAPIRequest(ctx, token, "https://api.github.com/user")
	if err != nil {
		return "", "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var user struct {
		Login string `json:"login"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return user.Login, user.Name, nil
}

func (g *GitHubProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	resp, err := g.makeGitHubAPIRequest(ctx, token, "https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to check token scopes: %w", err)
	}
	defer resp.Body.Close()

	// GitHub returns OAuth scopes in the X-OAuth-Scopes header
	scopesHeader := resp.Header.Get("X-OAuth-Scopes")
	if scopesHeader == "" {
		return []string{}, nil
	}

	// Parse comma-separated scopes
	scopes := []string{}
	for _, scope := range strings.Split(scopesHeader, ",") {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			scopes = append(scopes, scope)
		}
	}

	return scopes, nil
}
