package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	RegisterProvider("gitlab", Registration{
		New: func(cfg Config) Provider {
			return &GitLabProvider{
				host:     cfg.Host,
				clientID: cfg.ClientID,
			}
		},
		Detect:      NewGitLabProviderForHost,
		DefaultHost: "gitlab.com",
	})
}

// NewGitLabProviderForHost attempts to create a GitLab provider for the given host
// Returns nil, nil if the host is not a GitLab instance
// Returns nil, error if there was a network error during detection
func NewGitLabProviderForHost(ctx context.Context, client *http.Client, host string) (Provider, error) {
	// Known GitLab host
	if strings.ToLower(host) == "gitlab.com" {
		p := &GitLabProvider{host: host}
		return p, nil
	}

	// For other hosts, check if it's a GitLab instance using the version endpoint
	baseURL := fmt.Sprintf("https://%s", host)
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v4/version", baseURL), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, nil // Not a GitLab instance
		}
		// GitLab version endpoint returns version and revision
		if _, ok := data["version"]; ok {
			p := &GitLabProvider{host: host}
			return p, nil
		}
	}

	return nil, nil // Not a GitLab instance
}

type GitLabProvider struct {
	host     string
	clientID string
}

// getBaseURL returns the base URL for API calls
func (g *GitLabProvider) getBaseURL() string {
	if g.host != "" && g.host != "gitlab.com" {
		return fmt.Sprintf("https://%s", g.host)
	}
	return "https://gitlab.com"
}

// GitLab OAuth device flow response structures
type gitLabDeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type gitLabTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type gitLabErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// makeGitLabAPIRequest is a helper function to make authenticated requests to GitLab API
func (g *GitLabProvider) makeGitLabAPIRequest(ctx context.Context, token string, endpoint string) (*http.Response, error) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	return makeAuthenticatedRequest(ctx, "GET", endpoint, "Bearer "+token, headers)
}

func (g *GitLabProvider) Name() string {
	return "gitlab"
}

func (g *GitLabProvider) Host() string {
	if g.host != "" {
		return g.host
	}
	return "gitlab.com"
}

func (g *GitLabProvider) GetScopes() []string {
	// read_api scope allows read access to the API, including private repositories
	return []string{"read_api", "read_repository"}
}

func (g *GitLabProvider) Authenticate(ctx context.Context) (string, error) {
	clientID := g.clientID
	if clientID == "" {
		if g.host == "gitlab.com" || g.host == "" {
			// FIXME: taken from https://gitlab.com/gitlab-org/cli/-/issues/1338
			clientID = "41d48f9422ebd655dd9cf2947d6979681dfaddc6d0c56f7628f6ada59559af1e"
		} else {
			// Provide instructions for creating an OAuth app
			fmt.Println("GitLab OAuth authentication requires a Client ID.")
			fmt.Println("\nTo create one:")
			fmt.Printf("1. Go to %s/-/profile/applications\n", g.getBaseURL())
			fmt.Println("2. Create a new application with:")
			fmt.Println("   - Name: nix-auth (or any name you prefer)")
			fmt.Println("   - Redirect URI: urn:ietf:wg:oauth:2.0:oob")
			fmt.Println("   - Confidential: ☐ (unchecked)")
			fmt.Println("   - Scopes: ☑ read_api")
			fmt.Println("3. Copy the Application ID")
			fmt.Println("\nThen run:")
			fmt.Printf("  nix-auth login gitlab --host %s --client-id <your-application-id>\n", g.host)
			fmt.Println("\nOr set the GITLAB_CLIENT_ID environment variable:")
			fmt.Println("  export GITLAB_CLIENT_ID=<your-application-id>")
			fmt.Printf("  nix-auth login gitlab --host %s\n", g.host)
			return "", fmt.Errorf("client ID required for GitLab self-hosted (use --client-id flag or GITLAB_CLIENT_ID env var)")
		}
	}

	// Start device flow
	deviceCode, err := g.requestDeviceCode(ctx, clientID)
	if err != nil {
		return "", fmt.Errorf("failed to request device code: %w", err)
	}

	DisplayDeviceCode(deviceCode.UserCode)
	DisplayURLAndOpenBrowser(deviceCode.VerificationURIComplete)
	ShowWaitingMessage()

	// Poll for token
	token, err := g.pollForToken(ctx, clientID, deviceCode)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return token, nil
}

func (g *GitLabProvider) requestDeviceCode(ctx context.Context, clientID string) (*gitLabDeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", strings.Join(g.GetScopes(), " "))

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/oauth/authorize_device", g.getBaseURL()), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp gitLabErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s: %s", errorResp.Error, errorResp.ErrorDescription)
	}

	var deviceCode gitLabDeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCode); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &deviceCode, nil
}

func (g *GitLabProvider) pollForToken(ctx context.Context, clientID string, deviceCode *gitLabDeviceCodeResponse) (string, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}

	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode.DeviceCode)

	client := &http.Client{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/oauth/token", g.getBaseURL()), strings.NewReader(data.Encode()))
			if err != nil {
				return "", err
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return "", err
			}

			if resp.StatusCode == http.StatusOK {
				var tokenResp gitLabTokenResponse
				if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
					resp.Body.Close()
					return "", fmt.Errorf("failed to decode token response: %w", err)
				}
				resp.Body.Close()
				return tokenResp.AccessToken, nil
			}

			var errorResp gitLabErrorResponse
			if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
				resp.Body.Close()
				return "", fmt.Errorf("unexpected response from token endpoint")
			}
			resp.Body.Close()

			switch errorResp.Error {
			case "authorization_pending":
				// User hasn't authorized yet, continue polling
				continue
			case "slow_down":
				// Increase polling interval
				ticker.Reset(interval + 5*time.Second)
				continue
			case "expired_token":
				return "", fmt.Errorf("device code expired, please try again")
			case "access_denied":
				return "", fmt.Errorf("access denied by user")
			default:
				return "", fmt.Errorf("%s: %s", errorResp.Error, errorResp.ErrorDescription)
			}
		}
	}
}

func (g *GitLabProvider) ValidateToken(ctx context.Context, token string) (ValidationStatus, error) {
	resp, err := g.makeGitLabAPIRequest(ctx, token, fmt.Sprintf("%s/api/v4/user", g.getBaseURL()))
	if err != nil {
		return ValidationStatusInvalid, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	return ValidationStatusValid, nil
}

func (g *GitLabProvider) GetUserInfo(ctx context.Context, token string) (username, fullName string, err error) {
	resp, err := g.makeGitLabAPIRequest(ctx, token, fmt.Sprintf("%s/api/v4/user", g.getBaseURL()))
	if err != nil {
		return "", "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var user struct {
		Username string `json:"username"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return user.Username, user.Name, nil
}

func (g *GitLabProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	// GitLab provides token info through a specific endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v4/personal_access_tokens/self", g.getBaseURL()), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check token info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("token is invalid or expired")
	}

	// If the endpoint is not available (404), try to parse from OAuth token info
	if resp.StatusCode == http.StatusNotFound {
		// For OAuth tokens, scopes might be included in the token response
		// but GitLab doesn't expose them via API, so we return what we requested
		return g.GetScopes(), nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tokenInfo struct {
		Scopes []string `json:"scopes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenInfo.Scopes, nil
}
