package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/numtide/nix-auth/internal/config"
	"github.com/numtide/nix-auth/internal/provider"
	"github.com/numtide/nix-auth/internal/ui"
	"github.com/spf13/cobra"
)

const (
	// tabPadding is the padding for tabwriter output.
	tabPadding = 2
)

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show the status of configured access tokens",
	Long:         `Display all configured access tokens and validate them with their respective providers.`,
	RunE:         runStatus,
	SilenceUsage: true,
}

func runStatus(_ *cobra.Command, _ []string) error {
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
		w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)

		// Detect provider from host
		prov, err := provider.Detect(ctx, host, "")
		if err != nil {
			// This should never happen as Detect always returns a provider
			// If we reach this, it's a programming error
			panic(fmt.Sprintf("impossible: Detect returned error for host %s: %v", host, err))
		}

		providerName := prov.Name()

		token, err := cfg.GetToken(host)
		if err != nil {
			_, _ = fmt.Fprintf(w, "  Provider\t%s\n", providerName)
			_, _ = fmt.Fprintf(w, "  Status\t%s\n", fmt.Sprintf("✗ Error: %v", err))
			_ = w.Flush()

			continue
		}

		_, _ = fmt.Fprintf(w, "  Provider\t%s\n", providerName)

		// Validate token and get user info
		var statusStr string

		validationStatus, validationErr := prov.ValidateToken(ctx, token)

		switch validationStatus {
		case provider.ValidationStatusValid:
			statusStr = "✓ Valid"
			// Get user info if valid
			username, fullName, err := prov.GetUserInfo(ctx, token)
			if err == nil {
				if fullName != "" {
					_, _ = fmt.Fprintf(w, "  User\t%s (%s)\n", username, fullName)
				} else {
					_, _ = fmt.Fprintf(w, "  User\t%s\n", username)
				}
			}
		case provider.ValidationStatusInvalid:
			if validationErr != nil {
				statusStr = fmt.Sprintf("✗ Invalid - %v", validationErr)
			} else {
				statusStr = "✗ Invalid"
			}
		case provider.ValidationStatusUnknown:
			statusStr = "⚠ Unknown (unverified)"
		}

		// Mask token for security
		maskedToken := ui.MaskToken(token)
		_, _ = fmt.Fprintf(w, "  Token\t%s\n", maskedToken)

		// Show token scopes
		scopes, err := prov.GetTokenScopes(ctx, token)

		switch {
		case err != nil:
			_, _ = fmt.Fprintf(w, "  Scopes\tUnable to retrieve\n")
		case len(scopes) == 0:
			_, _ = fmt.Fprintf(w, "  Scopes\tNone\n")
		default:
			_, _ = fmt.Fprintf(w, "  Scopes\t%s\n", strings.Join(scopes, ", "))
		}

		_, _ = fmt.Fprintf(w, "  Status\t%s\n", statusStr)
		_ = w.Flush()
	}

	return nil
}
