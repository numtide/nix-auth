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
	Register("forgejo", &ForgejoProvider{})
	Register("codeberg", &ForgejoProvider{host: "codeberg.org"})
}

type ForgejoProvider struct {
	host string
}

func (f *ForgejoProvider) SetHost(host string) {
	f.host = host
}

func (f *ForgejoProvider) SetClientID(clientID string) {
}

func (f *ForgejoProvider) getBaseURL() string {
	if f.host != "" {
		return fmt.Sprintf("https://%s", f.host)
	}
	// This should not happen as we validate in Authenticate()
	return ""
}

func (f *ForgejoProvider) getAPIURL() string {
	return fmt.Sprintf("%s/api/v1", f.getBaseURL())
}

func (f *ForgejoProvider) makeForgejoAPIRequest(ctx context.Context, token string, endpoint string) (*http.Response, error) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	return makeAuthenticatedRequest(ctx, "GET", endpoint, "token "+token, headers)
}

func (f *ForgejoProvider) Name() string {
	return "forgejo"
}

func (f *ForgejoProvider) Host() string {
	return f.host
}

func (f *ForgejoProvider) GetScopes() []string {
	return []string{"read:repository", "read:user"}
}

func (f *ForgejoProvider) Authenticate(ctx context.Context) (string, error) {
	// Validate that we have a host
	if f.host == "" {
		return "", fmt.Errorf("--host flag is required for forgejo provider (e.g., --host git.company.com)")
	}

	fmt.Println()
	fmt.Println("Forgejo does not support OAuth device flow. You'll need to create a Personal Access Token.")
	fmt.Println()
	fmt.Println("Instructions:")
	fmt.Printf("1. Go to %s/user/settings/applications\n", f.getBaseURL())
	fmt.Println("2. In the 'Generate New Token' section, enter a token name (e.g., 'nix-auth')")
	fmt.Println("3. Select the following access and permissions:")
	fmt.Println("   - Repository and Organization Access: All (public, private, and limited)")
	fmt.Println("   - Permissions: read:repository, read:user")
	fmt.Println("4. Click 'Generate Token'")
	fmt.Println("5. Copy the generated token")
	fmt.Println()

	tokenURL := fmt.Sprintf("%s/user/settings/applications", f.getBaseURL())
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

	if err := f.ValidateToken(ctx, token); err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return token, nil
}

func (f *ForgejoProvider) ValidateToken(ctx context.Context, token string) error {
	userURL := fmt.Sprintf("%s/user", f.getAPIURL())
	resp, err := f.makeForgejoAPIRequest(ctx, token, userURL)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (f *ForgejoProvider) GetUserInfo(ctx context.Context, token string) (username, fullName string, err error) {
	userURL := fmt.Sprintf("%s/user", f.getAPIURL())
	resp, err := f.makeForgejoAPIRequest(ctx, token, userURL)
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

func (f *ForgejoProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	return f.GetScopes(), nil
}