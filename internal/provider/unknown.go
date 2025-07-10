package provider

import (
	"context"
	"fmt"

	"github.com/numtide/nix-auth/internal/ui"
)

// NewUnknownProvider creates a new instance of UnknownProvider.
func NewUnknownProvider(host string) *UnknownProvider {
	return &UnknownProvider{
		host: host,
	}
}

// UnknownProvider handles hosts that don't match any known provider.
type UnknownProvider struct {
	host string
}

// Name returns the provider name "unknown".
func (u *UnknownProvider) Name() string {
	return "unknown"
}

// Host returns the hostname for this unknown provider.
func (u *UnknownProvider) Host() string {
	return u.host
}

// GetScopes returns an empty list as scopes are unknown.
func (u *UnknownProvider) GetScopes() []string {
	return []string{}
}

// Authenticate prompts the user to manually enter a token for an unknown provider.
func (u *UnknownProvider) Authenticate(_ context.Context) (string, error) {
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

// ValidateToken always returns unknown status as validation is not possible.
func (u *UnknownProvider) ValidateToken(_ context.Context, _ string) (ValidationStatus, error) {
	// Unknown providers cannot validate tokens
	return ValidationStatusUnknown, nil
}

// GetUserInfo returns an error as user info is not available for unknown providers.
func (u *UnknownProvider) GetUserInfo(_ context.Context, _ string) (username, fullName string, err error) {
	return "", "", fmt.Errorf("user info not available for unknown provider")
}

// GetTokenScopes returns an empty list as scopes cannot be determined for unknown providers.
func (u *UnknownProvider) GetTokenScopes(_ context.Context, _ string) ([]string, error) {
	// Return empty scopes for unknown providers
	return []string{}, nil
}
