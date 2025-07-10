package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/numtide/nix-auth/internal/nixconf"
	"github.com/numtide/nix-auth/internal/provider"
	"github.com/numtide/nix-auth/internal/ui"
	"github.com/spf13/cobra"
)

const (
	// tabPadding is the padding for tabwriter output.
	tabPadding = 2
)

var statusCmd = &cobra.Command{
	Use:   "status [host...]",
	Short: "Show the status of configured access tokens",
	Long: `Display all configured access tokens and validate them with their respective providers.

If no hosts are specified, all configured tokens are shown.
If one or more hosts are specified, only tokens for those hosts are displayed.`,
	RunE:         runStatus,
	SilenceUsage: true,
}

func runStatus(_ *cobra.Command, args []string) error {
	cfg, err := nixconf.New(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	hosts, err := getHostsToShow(cfg, args)
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		return showNoTokensMessage(cfg)
	}

	showHeader(hosts, args, cfg)

	ctx := context.Background()

	for i, host := range hosts {
		if i > 0 {
			fmt.Println()
		}

		showHostStatus(ctx, host, cfg)
	}

	return nil
}

// getHostsToShow returns the list of hosts to display status for.
func getHostsToShow(cfg *nixconf.NixConfig, args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}

	hosts, err := cfg.ListTokens()
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}

	return hosts, nil
}

// showNoTokensMessage displays a message when no tokens are configured.
func showNoTokensMessage(cfg *nixconf.NixConfig) error {
	fmt.Println("No access tokens configured.")
	fmt.Printf("Config file: %s\n", cfg.GetPath())
	fmt.Println("\nRun 'nix-auth login' to add a token.")

	return nil
}

// showHeader displays the header for the status output.
func showHeader(hosts []string, args []string, cfg *nixconf.NixConfig) {
	if len(args) > 0 {
		fmt.Printf("Access Tokens (showing %d hosts from %s)\n\n", len(hosts), cfg.GetPath())
	} else {
		fmt.Printf("Access Tokens (%d configured in %s)\n\n", len(hosts), cfg.GetPath())
	}
}

// showHostStatus displays the status information for a single host.
func showHostStatus(ctx context.Context, host string, cfg *nixconf.NixConfig) {
	fmt.Printf("%s\n", host)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	defer func() { _ = w.Flush() }()

	prov, err := provider.Detect(ctx, host, "")
	if err != nil {
		panic(fmt.Sprintf("impossible: Detect returned error for host %s: %v", host, err))
	}

	providerName := prov.Name()

	token, err := cfg.GetToken(host)
	if err != nil {
		showTokenError(w, providerName, err)
		return
	}

	if token == "" {
		showNoTokenConfigured(w, providerName)
		return
	}

	showTokenDetails(ctx, w, prov, providerName, token)
}

// showTokenError displays an error when getting a token fails.
func showTokenError(w *tabwriter.Writer, providerName string, err error) {
	_, _ = fmt.Fprintf(w, "  Provider\t%s\n", providerName)
	_, _ = fmt.Fprintf(w, "  Status\t%s\n", fmt.Sprintf("✗ Error: %v", err))
}

// showNoTokenConfigured displays a message when no token is configured for a host.
func showNoTokenConfigured(w *tabwriter.Writer, providerName string) {
	_, _ = fmt.Fprintf(w, "  Provider\t%s\n", providerName)
	_, _ = fmt.Fprintf(w, "  Status\t✗ No token configured\n")
}

// showTokenDetails displays detailed information about a token.
func showTokenDetails(ctx context.Context, w *tabwriter.Writer, prov provider.Provider, providerName, token string) {
	_, _ = fmt.Fprintf(w, "  Provider\t%s\n", providerName)

	statusStr := getValidationStatus(ctx, prov, token, w)

	maskedToken := ui.MaskToken(token)
	_, _ = fmt.Fprintf(w, "  Token\t%s\n", maskedToken)

	showTokenScopes(ctx, w, prov, token)

	_, _ = fmt.Fprintf(w, "  Status\t%s\n", statusStr)
}

// getValidationStatus validates a token and returns the status string.
func getValidationStatus(ctx context.Context, prov provider.Provider, token string, w *tabwriter.Writer) string {
	validationStatus, validationErr := prov.ValidateToken(ctx, token)

	switch validationStatus {
	case provider.ValidationStatusValid:
		showUserInfo(ctx, prov, token, w)
		return "✓ Valid"
	case provider.ValidationStatusInvalid:
		if validationErr != nil {
			return fmt.Sprintf("✗ Invalid - %v", validationErr)
		}

		return "✗ Invalid"
	case provider.ValidationStatusUnknown:
		return "⚠ Unknown (unverified)"
	default:
		return "⚠ Unknown"
	}
}

// showUserInfo displays user information if available.
func showUserInfo(ctx context.Context, prov provider.Provider, token string, w *tabwriter.Writer) {
	username, fullName, err := prov.GetUserInfo(ctx, token)
	if err == nil {
		if fullName != "" {
			_, _ = fmt.Fprintf(w, "  User\t%s (%s)\n", username, fullName)
		} else {
			_, _ = fmt.Fprintf(w, "  User\t%s\n", username)
		}
	}
}

// showTokenScopes displays the token scopes.
func showTokenScopes(ctx context.Context, w *tabwriter.Writer, prov provider.Provider, token string) {
	scopes, err := prov.GetTokenScopes(ctx, token)

	switch {
	case err != nil:
		_, _ = fmt.Fprintf(w, "  Scopes\tUnable to retrieve\n")
	case len(scopes) == 0:
		_, _ = fmt.Fprintf(w, "  Scopes\tNone\n")
	default:
		_, _ = fmt.Fprintf(w, "  Scopes\t%s\n", strings.Join(scopes, ", "))
	}
}
