package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zimbatm/nix-auth/internal/config"
	"github.com/zimbatm/nix-auth/internal/provider"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of configured access tokens",
	Long:  `Display all configured access tokens and validate them with their respective providers.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	hosts, err := cfg.ListTokens()
	if err != nil {
		return fmt.Errorf("failed to list tokens: %w", err)
	}

	if len(hosts) == 0 {
		fmt.Println("No access tokens configured.")
		fmt.Println("\nRun 'nix-auth login' to add a token.")
		return nil
	}

	fmt.Printf("Found %d configured token(s):\n\n", len(hosts))

	ctx := context.Background()
	for _, host := range hosts {
		fmt.Printf("Host: %s\n", host)

		// Try to determine provider from host
		var prov provider.Provider
		for _, p := range provider.Registry {
			if strings.Contains(host, p.Host()) {
				prov = p
				break
			}
		}

		if prov == nil {
			fmt.Printf("  Status: Unknown provider\n")
			fmt.Printf("  Token: Configured (validation not available)\n\n")
			continue
		}

		token, err := cfg.GetToken(host)
		if err != nil {
			fmt.Printf("  Status: Error reading token: %v\n\n", err)
			continue
		}

		fmt.Printf("  Provider: %s\n", prov.Name())
		fmt.Print("  Status: ")

		if err := prov.ValidateToken(ctx, token); err != nil {
			fmt.Printf("Invalid - %v\n", err)
		} else {
			fmt.Printf("Valid\n")
		}

		// Mask token for security
		if len(token) > 10 {
			fmt.Printf("  Token: %s...%s\n", token[:4], token[len(token)-4:])
		} else {
			fmt.Printf("  Token: Configured\n")
		}
		fmt.Println()
	}

	return nil
}
