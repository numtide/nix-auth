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

type GitHubProvider struct {
	host     string
	clientID string
}

// SetHost sets a custom host for the GitHub provider
func (g *GitHubProvider) SetHost(host string) {
	g.host = host
}

// SetClientID sets a custom OAuth client ID for the GitHub provider
func (g *GitHubProvider) SetClientID(clientID string) {
	g.clientID = clientID
}

// getBaseURL returns the base URL for web URLs
func (g *GitHubProvider) getBaseURL() string {
	if g.host != "" && g.host != "github.com" {
		return fmt.Sprintf("https://%s", g.host)
	}
	return "https://github.com"
}

// getAPIURL returns the base URL for API calls
func (g *GitHubProvider) getAPIURL() string {
	if g.host != "" && g.host != "github.com" {
		// GitHub Enterprise uses {host}/api/v3
		return fmt.Sprintf("https://%s/api/v3", g.host)
	}
	// GitHub.com uses api.github.com
	return "https://api.github.com"
}

// makeGitHubAPIRequest is a helper function to make authenticated requests to GitHub API
func (g *GitHubProvider) makeGitHubAPIRequest(ctx context.Context, token string, endpoint string) (*http.Response, error) {
	headers := map[string]string{
		"Accept": "application/vnd.github.v3+json",
	}
	return makeAuthenticatedRequest(ctx, "GET", endpoint, "token "+token, headers)
}

func (g *GitHubProvider) Name() string {
	return "github"
}

func (g *GitHubProvider) Host() string {
	if g.host != "" {
		return g.host
	}
	return "github.com"
}

func (g *GitHubProvider) GetScopes() []string {
	// Minimal scope needed for private repo access
	return []string{"repo"}
}

func (g *GitHubProvider) Authenticate(ctx context.Context) (string, error) {
	clientID := g.clientID
	if clientID == "" {
		if g.host == "github.com" || g.host == "" {
			clientID = "178c6fc778ccc68e1d6a" // GitHub CLI's client ID - widely used for CLI tools
		} else {
			// Provide instructions for creating an OAuth app
			fmt.Println("GitHub Enterprise OAuth authentication requires a Client ID.")
			fmt.Println("\nTo create one:")
			fmt.Printf("1. Go to %s/settings/applications/new\n", g.getBaseURL())
			fmt.Println("2. Create a new OAuth App with:")
			fmt.Println("   - Application name: nix-auth (or any name you prefer)")
			fmt.Println("   - Homepage URL: https://github.com/numtide/nix-auth")
			fmt.Println("   - Authorization callback URL: http://127.0.0.1/callback")
			fmt.Println("3. After creating, copy the Client ID")
			fmt.Println("\nThen run:")
			fmt.Printf("  nix-auth login github --host %s --client-id <your-client-id>\n", g.host)
			fmt.Println("\nOr set the GITHUB_CLIENT_ID environment variable:")
			fmt.Println("  export GITHUB_CLIENT_ID=<your-client-id>")
			fmt.Printf("  nix-auth login github --host %s\n", g.host)
			return "", fmt.Errorf("client ID required for GitHub Enterprise (use --client-id flag or GITHUB_CLIENT_ID env var)")
		}
	}

	scopes := g.GetScopes()
	httpClient := &http.Client{}

	// Request device code
	deviceCodeURL := fmt.Sprintf("%s/login/device/code", g.getBaseURL())
	code, err := device.RequestCode(httpClient, deviceCodeURL, clientID, scopes)
	if err != nil {
		return "", fmt.Errorf("failed to request device code: %w", err)
	}

	DisplayDeviceCode(code.UserCode)
	DisplayURLAndOpenBrowser(code.VerificationURI)
	ShowWaitingMessage()

	// Wait for user to authorize
	accessTokenURL := fmt.Sprintf("%s/login/oauth/access_token", g.getBaseURL())
	accessToken, err := device.Wait(ctx, httpClient, accessTokenURL, device.WaitOptions{
		ClientID:   clientID,
		DeviceCode: code,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return accessToken.Token, nil
}

func (g *GitHubProvider) ValidateToken(ctx context.Context, token string) error {
	userURL := fmt.Sprintf("%s/user", g.getAPIURL())
	resp, err := g.makeGitHubAPIRequest(ctx, token, userURL)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (g *GitHubProvider) GetUserInfo(ctx context.Context, token string) (username, fullName string, err error) {
	userURL := fmt.Sprintf("%s/user", g.getAPIURL())
	resp, err := g.makeGitHubAPIRequest(ctx, token, userURL)
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
	userURL := fmt.Sprintf("%s/user", g.getAPIURL())
	resp, err := g.makeGitHubAPIRequest(ctx, token, userURL)
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
