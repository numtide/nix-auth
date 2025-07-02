package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cli/browser"
	"github.com/cli/oauth/device"
)

func init() {
	Register("github", &GitHubProvider{})
}

type GitHubProvider struct{}

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

	fmt.Printf("First, copy your one-time code: %s\n", code.UserCode)
	fmt.Printf("Then press Enter to open github.com in your browser...\n")
	fmt.Scanln()

	// Open browser
	if err := browser.OpenURL(code.VerificationURI); err != nil {
		fmt.Printf("Failed to open browser. Please visit: %s\n", code.VerificationURI)
	}

	fmt.Println("Waiting for authorization...")

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
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("token is invalid or expired")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse user info
	var user struct {
		Login string `json:"login"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("Authenticated as: %s", user.Login)
	if user.Name != "" {
		fmt.Printf(" (%s)", user.Name)
	}
	fmt.Println()

	return nil
}
