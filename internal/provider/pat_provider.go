package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cli/browser"
)

// PersonalAccessTokenProvider provides common functionality for providers that use Personal Access Tokens
type PersonalAccessTokenProvider struct {
	host         string
	providerName string
	defaultHost  string
}

func (p *PersonalAccessTokenProvider) SetHost(host string) {
	p.host = host
}

func (p *PersonalAccessTokenProvider) SetClientID(clientID string) {
	// Personal Access Token providers don't use client IDs
}

func (p *PersonalAccessTokenProvider) Name() string {
	return p.providerName
}

func (p *PersonalAccessTokenProvider) Host() string {
	if p.host != "" {
		return p.host
	}
	return p.defaultHost
}

func (p *PersonalAccessTokenProvider) GetScopes() []string {
	return []string{"read:repository", "read:user"}
}

func (p *PersonalAccessTokenProvider) getBaseURL() string {
	host := p.Host()
	if host != "" {
		return fmt.Sprintf("https://%s", host)
	}
	return ""
}

func (p *PersonalAccessTokenProvider) getAPIURL() string {
	return fmt.Sprintf("%s/api/v1", p.getBaseURL())
}

func (p *PersonalAccessTokenProvider) makeAPIRequest(ctx context.Context, token string, endpoint string) (*http.Response, error) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	return makeAuthenticatedRequest(ctx, "GET", endpoint, "token "+token, headers)
}

func (p *PersonalAccessTokenProvider) Authenticate(ctx context.Context) (string, error) {
	// Validate that we have a host
	if p.Host() == "" {
		return "", fmt.Errorf("--host flag is required for %s provider (e.g., --host git.company.com)", p.providerName)
	}

	fmt.Println()
	fmt.Printf("%s does not support OAuth device flow. You'll need to create a Personal Access Token.\n", strings.Title(p.providerName))
	fmt.Println()
	fmt.Println("Instructions:")
	fmt.Printf("1. Go to %s/user/settings/applications\n", p.getBaseURL())
	fmt.Println("2. In the 'Generate New Token' section, enter a token name (e.g., 'nix-auth')")
	fmt.Println("3. Select the following access and permissions:")
	fmt.Println("   - Repository and Organization Access: All (public, private, and limited)")
	fmt.Println("   - Permissions: read:repository, read:user")
	fmt.Println("4. Click 'Generate Token'")
	fmt.Println("5. Copy the generated token")
	fmt.Println()

	tokenURL := fmt.Sprintf("%s/user/settings/applications", p.getBaseURL())
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

	if err := p.ValidateToken(ctx, token); err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return token, nil
}

func (p *PersonalAccessTokenProvider) ValidateToken(ctx context.Context, token string) error {
	userURL := fmt.Sprintf("%s/user", p.getAPIURL())
	resp, err := p.makeAPIRequest(ctx, token, userURL)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (p *PersonalAccessTokenProvider) GetUserInfo(ctx context.Context, token string) (username, fullName string, err error) {
	userURL := fmt.Sprintf("%s/user", p.getAPIURL())
	resp, err := p.makeAPIRequest(ctx, token, userURL)
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

func (p *PersonalAccessTokenProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	return p.GetScopes(), nil
}
