package provider

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
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

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Would you like to manually add a token for this host? [y/N] ")

	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		return "", fmt.Errorf("login cancelled")
	}

	fmt.Println("\nPlease enter your personal access token.")
	fmt.Println("This token will be saved but cannot be validated automatically.")
	fmt.Print("\nToken: ")

	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

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
