package cmd

import (
	"fmt"
	"strings"

	"github.com/numtide/nix-auth/internal/config"
	"github.com/numtide/nix-auth/internal/provider"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout [provider|host]",
	Short: "Remove an access token",
	Long: `Remove an access token from your nix.conf.
You can specify either a provider name (github, gitlab) or a full host.`,
	Example: `  nix-auth logout github
  nix-auth logout github.com
  nix-auth logout gitlab.company.com`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.New(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	if len(args) == 0 {
		// Interactive mode - list tokens and ask which to remove
		hosts, err := cfg.ListTokens()
		if err != nil {
			return fmt.Errorf("failed to list tokens: %w", err)
		}

		if len(hosts) == 0 {
			fmt.Println("No access tokens configured.")
			return nil
		}

		fmt.Println("Select a token to remove:")
		for i, host := range hosts {
			fmt.Printf("  %d. %s\n", i+1, host)
		}
		fmt.Print("\nEnter number (or 0 to cancel): ")

		var choice int
		fmt.Scanln(&choice)

		if choice == 0 || choice > len(hosts) {
			fmt.Println("Logout cancelled.")
			return nil
		}

		return removeToken(cfg, hosts[choice-1])
	}

	// Determine host from argument
	arg := strings.ToLower(args[0])

	// Check if it's a provider name
	if prov, ok := provider.Get(arg); ok {
		return removeToken(cfg, prov.Host())
	}

	// Otherwise treat it as a host
	return removeToken(cfg, arg)
}

func removeToken(cfg *config.NixConfig, host string) error {
	fmt.Printf("Removing token for %s...\n", host)

	if err := cfg.RemoveToken(host); err != nil {
		return fmt.Errorf("failed to remove token: %w", err)
	}

	fmt.Printf("✓ Successfully removed token for %s\n", host)
	return nil
}
