package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/numtide/nix-auth/internal/config"
	"github.com/numtide/nix-auth/internal/provider"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Authenticate with a provider and save the access token",
	Long: `Authenticate with a provider (GitHub, GitLab, etc.) using OAuth device flow
and save the access token to your nix.conf for use with Nix flakes.`,
	Example: `  nix-auth login                      # defaults to GitHub
  nix-auth login github
  nix-auth login gitlab
  nix-auth login github --host github.company.com --client-id abc123`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogin,
}

var (
	loginHost     string
	loginClientID string
)

func init() {
	loginCmd.Flags().StringVar(&loginHost, "host", "", "Custom host (e.g., github.company.com)")
	loginCmd.Flags().StringVar(&loginClientID, "client-id", "", "OAuth client ID (required for self-hosted instances)")
}

func runLogin(cmd *cobra.Command, args []string) error {
	providerName := "github" // default
	if len(args) > 0 {
		providerName = strings.ToLower(args[0])
	}

	// Get provider
	prov, ok := provider.Get(providerName)
	if !ok {
		available := strings.Join(provider.List(), ", ")
		return fmt.Errorf("unknown provider '%s'. Available providers: %s", providerName, available)
	}

	// Determine host
	host := prov.Host()
	if loginHost != "" {
		host = loginHost
	}

	// Always set the host (even if it's the default)
	prov.SetHost(host)

	// Set client ID: use flag, fallback to environment variable
	clientID := loginClientID
	if clientID == "" {
		// Check provider-specific environment variable
		switch providerName {
		case "github":
			clientID = os.Getenv("GITHUB_CLIENT_ID")
		case "gitlab":
			clientID = os.Getenv("GITLAB_CLIENT_ID")
		}
	}
	if clientID != "" {
		prov.SetClientID(clientID)
	}

	fmt.Printf("Authenticating with %s (%s)...\n", prov.Name(), host)

	// Check if token already exists
	cfg, err := config.New(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	existingToken, _ := cfg.GetToken(host)
	if existingToken != "" {
		fmt.Printf("A token for %s already exists. Do you want to replace it? [y/N] ", host)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Login cancelled.")
			return nil
		}
	}

	// Perform authentication
	ctx := context.Background()
	token, err := prov.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Validate token
	fmt.Println("\nValidating token...")
	if err := prov.ValidateToken(ctx, token); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Save token
	if err := cfg.SetToken(host, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("\nSuccessfully authenticated and saved token for %s\n", host)
	fmt.Printf("Token saved to: %s\n", cfg.GetPath())

	return nil
}
