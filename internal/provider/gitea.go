package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cli/browser"
)

func init() {
	Register("gitea", &GiteaProvider{})
}

type GiteaProvider struct {
	host string
}

func (g *GiteaProvider) SetHost(host string) {
	g.host = host
}

func (g *GiteaProvider) SetClientID(clientID string) {
}

func (g *GiteaProvider) getBaseURL() string {
	if g.host != "" {
		return fmt.Sprintf("https://%s", g.host)
	}
	return "https://gitea.com"
}

func (g *GiteaProvider) getAPIURL() string {
	return fmt.Sprintf("%s/api/v1", g.getBaseURL())
}

func (g *GiteaProvider) makeGiteaAPIRequest(ctx context.Context, token string, endpoint string) (*http.Response, error) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	return makeAuthenticatedRequest(ctx, "GET", endpoint, "token "+token, headers)
}

func (g *GiteaProvider) Name() string {
	return "gitea"
}

func (g *GiteaProvider) Host() string {
	if g.host != "" {
		return g.host
	}
	return "gitea.com"
}

func (g *GiteaProvider) GetScopes() []string {
	return []string{"read:repository", "read:user"}
}

func (g *GiteaProvider) Authenticate(ctx context.Context) (string, error) {
	fmt.Println()
	fmt.Println("Gitea does not support OAuth device flow. You'll need to create a Personal Access Token.")
	fmt.Println()
	fmt.Println("Instructions:")
	fmt.Printf("1. Go to %s/user/settings/applications\n", g.getBaseURL())
	fmt.Println("2. In the 'Generate New Token' section, enter a token name (e.g., 'nix-auth')")
	fmt.Println("3. Select the following access and permissions:")
	fmt.Println("   - Repository and Organization Access: All (public, private, and limited)")
	fmt.Println("   - Permissions: read:repository, read:user")
	fmt.Println("4. Click 'Generate Token'")
	fmt.Println("5. Copy the generated token")
	fmt.Println()

	tokenURL := fmt.Sprintf("%s/user/settings/applications", g.getBaseURL())
	fmt.Printf("Opening %s in your browser...\n", tokenURL)

	if err := browser.OpenURL(tokenURL); err != nil {
		fmt.Println("Could not open browser automatically.")
		fmt.Printf("Please manually visit: %s\n", tokenURL)
	}

	fmt.Println()
	var token string
	fmt.Print("Enter your Personal Access Token: ")
	if _, err := fmt.Scanln(&token); err != nil {
		return "", fmt.Errorf("failed to read token: %w", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	if err := g.ValidateToken(ctx, token); err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return token, nil
}

func (g *GiteaProvider) ValidateToken(ctx context.Context, token string) error {
	userURL := fmt.Sprintf("%s/user", g.getAPIURL())
	resp, err := g.makeGiteaAPIRequest(ctx, token, userURL)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (g *GiteaProvider) GetUserInfo(ctx context.Context, token string) (username, fullName string, err error) {
	userURL := fmt.Sprintf("%s/user", g.getAPIURL())
	resp, err := g.makeGiteaAPIRequest(ctx, token, userURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var user struct {
		Login    string `json:"login"`
		Username string `json:"username"`
		FullName string `json:"full_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	username = user.Username
	if username == "" {
		username = user.Login
	}

	return username, user.FullName, nil
}

func (g *GiteaProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	return g.GetScopes(), nil
}
