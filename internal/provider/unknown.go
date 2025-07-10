package provider

import (
	"context"
	"fmt"

	"github.com/numtide/nix-auth/internal/ui"
)

// NewUnknownProvider creates a new instance of UnknownProvider
func NewUnknownProvider(host string) *UnknownProvider {
	return &UnknownProvider{
		host: host,
	}
}

// UnknownProvider handles hosts that don't match any known provider
type UnknownProvider struct {
	host  string
	token string
}

func (u *UnknownProvider) Name() string {
	return "unknown"
}

func (u *UnknownProvider) Host() string {
	return u.host
}

func (u *UnknownProvider) GetScopes() []string {
	return []string{}
}

func (u *UnknownProvider) Authenticate(ctx context.Context) (string, error) {
	fmt.Printf("Unable to auto-detect provider type for %s\n\n", u.host)

	confirm, err := ui.ReadYesNo("Would you like to manually add a token for this host? [y/N] ")
	if err != nil {
		return "", fmt.Errorf("failed to read confirmation: %w", err)
	}

	if !confirm {
		return "", fmt.Errorf("login cancelled")
	}

	fmt.Println("\nPlease enter your personal access token.")
	fmt.Println("This token will be saved but cannot be validated automatically.")
	fmt.Println()

	token, err := ui.ReadSecureInput("Token: ")
	if err != nil {
		return "", fmt.Errorf("failed to read token: %w", err)
	}

	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	return token, nil
}

func (u *UnknownProvider) ValidateToken(ctx context.Context, token string) (ValidationStatus, error) {
	// Unknown providers cannot validate tokens
	return ValidationStatusUnknown, nil
}

func (u *UnknownProvider) GetUserInfo(ctx context.Context, token string) (username, fullName string, err error) {
	return "", "", fmt.Errorf("user info not available for unknown provider")
}

func (u *UnknownProvider) GetTokenScopes(ctx context.Context, token string) ([]string, error) {
	// Return empty scopes for unknown providers
	return []string{}, nil
}
