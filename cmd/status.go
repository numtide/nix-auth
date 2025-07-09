package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/numtide/nix-auth/internal/config"
	"github.com/numtide/nix-auth/internal/provider"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show the status of configured access tokens",
	Long:         `Display all configured access tokens and validate them with their respective providers.`,
	RunE:         runStatus,
	SilenceUsage: true,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.New(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	hosts, err := cfg.ListTokens()
	if err != nil {
		return fmt.Errorf("failed to list tokens: %w", err)
	}

	if len(hosts) == 0 {
		fmt.Println("No access tokens configured.")
		fmt.Printf("Config file: %s\n", cfg.GetPath())
		fmt.Println("\nRun 'nix-auth login' to add a token.")
		return nil
	}

	fmt.Printf("Access Tokens (%d configured in %s)\n\n", len(hosts), cfg.GetPath())

	ctx := context.Background()
	for i, host := range hosts {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("%s\n", host)

		// Create a tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		// Detect provider from host
		var prov provider.Provider
		providerName := ""

		detectedProvider, err := provider.DetectProviderFromHost(ctx, host)
		if err == nil {
			if p, ok := provider.Get(detectedProvider); ok {
				prov = p
				providerName = detectedProvider
				// Set the host on the provider instance
				prov.SetHost(host)
			}
		}

		if prov == nil {
			fmt.Fprintf(w, "  Provider\t%s\n", "unknown")
			fmt.Fprintf(w, "  Status\t%s\n", "✗ Unknown provider")
			fmt.Fprintf(w, "  Token\t%s\n", "Configured (validation not available)")
			w.Flush()
			continue
		}

		token, err := cfg.GetToken(host)
		if err != nil {
			fmt.Fprintf(w, "  Provider\t%s\n", providerName)
			fmt.Fprintf(w, "  Status\t%s\n", fmt.Sprintf("✗ Error: %v", err))
			w.Flush()
			continue
		}

		fmt.Fprintf(w, "  Provider\t%s\n", providerName)

		// Validate token and get user info
		var statusStr string
		validationErr := prov.ValidateToken(ctx, token)
		if validationErr != nil {
			statusStr = fmt.Sprintf("✗ Invalid - %v", validationErr)
		} else {
			statusStr = "✓ Valid"

			// Get user info if valid
			username, fullName, err := prov.GetUserInfo(ctx, token)
			if err == nil {
				if fullName != "" {
					fmt.Fprintf(w, "  User\t%s (%s)\n", username, fullName)
				} else {
					fmt.Fprintf(w, "  User\t%s\n", username)
				}
			}
		}

		// Mask token for security
		var maskedToken string
		if len(token) > 10 {
			maskedToken = fmt.Sprintf("%s****%s", token[:4], token[len(token)-4:])
		} else {
			maskedToken = "Configured"
		}
		fmt.Fprintf(w, "  Token\t%s\n", maskedToken)

		// Show token scopes
		scopes, err := prov.GetTokenScopes(ctx, token)
		if err != nil {
			fmt.Fprintf(w, "  Scopes\tUnable to retrieve\n")
		} else if len(scopes) == 0 {
			fmt.Fprintf(w, "  Scopes\tNone\n")
		} else {
			fmt.Fprintf(w, "  Scopes\t%s\n", strings.Join(scopes, ", "))
		}

		fmt.Fprintf(w, "  Status\t%s\n", statusStr)
		w.Flush()
	}

	return nil
}
