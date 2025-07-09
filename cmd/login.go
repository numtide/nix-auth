package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/numtide/nix-auth/internal/config"
	"github.com/numtide/nix-auth/internal/provider"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [provider-or-host]",
	Short: "Authenticate with a provider and save the access token",
	Long: `Authenticate with a provider using OAuth device flow (or Personal Access Token for Gitea/Forgejo)
and save the access token to your nix.conf for use with Nix flakes.

You can specify either:
- A provider alias (github, gitlab, gitea, codeberg) - uses default host for that provider
- A host (e.g., github.com, git.company.com) - auto-detects provider type by querying API

Notes:
- The --provider flag only works when specifying a host, not with provider aliases
- For Forgejo, you must specify a host as it has no default: nix-auth login <host> --provider forgejo
- Using both a provider alias and --provider flag will result in an error`,
	SilenceUsage: true,
	Example: `  # Using provider aliases
  nix-auth login                           # defaults to github
  nix-auth login github
  nix-auth login gitlab
  nix-auth login gitea
  nix-auth login codeberg
  
  # Using hosts with auto-detection
  nix-auth login github.com
  nix-auth login gitlab.company.com        # auto-detects provider type
  nix-auth login git.company.com           # auto-detects provider type
  
  # Explicit provider specification
  nix-auth login git.company.com --provider forgejo
  nix-auth login github.company.com --client-id abc123`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogin,
}

var (
	loginProvider string
	loginClientID string
	loginForce    bool
	loginTimeout  int
	loginDryRun   bool
)

func init() {
	loginCmd.Flags().StringVar(&loginProvider, "provider", "auto", "Provider type when using a host (auto, github, gitlab, gitea, forgejo, codeberg)")
	loginCmd.Flags().StringVar(&loginClientID, "client-id", "", "OAuth client ID (required for GitHub Enterprise, optional for others)")
	loginCmd.Flags().BoolVar(&loginForce, "force", false, "Skip confirmation prompt when replacing existing tokens")
	loginCmd.Flags().IntVar(&loginTimeout, "timeout", 30, "Timeout in seconds for network operations")
	loginCmd.Flags().BoolVar(&loginDryRun, "dry-run", false, "Preview what would happen without authenticating")
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Parse the input
	input := "github" // default
	if len(args) > 0 {
		input = strings.ToLower(args[0])
	}

	// Resolve provider and host
	prov, host, err := resolveProviderAndHost(input, loginProvider, loginTimeout)
	if err != nil {
		return err
	}

	fmt.Printf("Authenticating with %s (%s)...\n", prov.Name(), host)

	// If dry-run, show what would happen and exit
	if loginDryRun {
		fmt.Println("\nDry-run mode: Preview of what would happen:")
		fmt.Printf("- Provider: %s\n", prov.Name())
		fmt.Printf("- Host: %s\n", host)
		fmt.Printf("- OAuth scopes: %s\n", strings.Join(prov.GetScopes(), ", "))
		if loginClientID != "" {
			fmt.Printf("- Client ID: %s\n", loginClientID)
		}
		fmt.Printf("- Config file: %s\n", configPath)
		fmt.Println("\nNo authentication performed. Run without --dry-run to authenticate.")
		return nil
	}

	// Check if token already exists
	cfg, err := config.New(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	existingToken, _ := cfg.GetToken(host)
	if existingToken != "" && !loginForce {
		fmt.Printf("A token for %s already exists. Do you want to replace it? [y/N] ", host)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Login cancelled.")
			return nil
		}
	}

	// Perform authentication
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(loginTimeout)*time.Second)
	defer cancel()
	token, err := prov.Authenticate(ctx)
	if err != nil {
		errMsg := fmt.Sprintf("authentication failed: %v", err)
		if strings.Contains(err.Error(), "context deadline exceeded") {
			errMsg += fmt.Sprintf("\n\nThe operation timed out after %d seconds. Try:\n"+
				"- Increasing the timeout: --timeout 60\n"+
				"- Checking your internet connection\n"+
				"- Verifying the host is accessible: curl https://%s", loginTimeout, host)
		} else if strings.Contains(err.Error(), "client ID") {
			errMsg += "\n\nFor self-hosted instances, you need to create an OAuth application.\n" +
				"See the instructions above or use --dry-run to preview the configuration."
		}
		return fmt.Errorf(errMsg)
	}

	// Validate token
	fmt.Println("\nValidating token...")
	status, err := prov.ValidateToken(ctx, token)
	if err != nil && status != provider.ValidationStatusUnknown {
		return fmt.Errorf("token validation failed: %w", err)
	}
	if status == provider.ValidationStatusInvalid {
		return fmt.Errorf("token is invalid")
	}
	if status == provider.ValidationStatusUnknown {
		fmt.Println("Warning: Token cannot be verified (unknown provider)")
	}

	// Save token
	if err := cfg.SetToken(host, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Printf("\nSuccessfully authenticated and saved token for %s\n", host)
	fmt.Printf("Token saved to: %s\n", cfg.GetPath())

	return nil
}

// resolveProviderAndHost determines the provider and host from the input
func resolveProviderAndHost(input, providerFlag string, timeout int) (provider.Provider, string, error) {
	// Check if input is a provider alias
	if reg, ok := provider.GetRegistration(input); ok {
		// It's a provider alias
		if providerFlag != "auto" && providerFlag != input {
			return nil, "", fmt.Errorf("cannot use --provider %s with provider alias '%s'\n"+
				"Use: nix-auth login %s", providerFlag, input, input)
		}

		host := reg.DefaultHost
		if host == "" {
			return nil, "", fmt.Errorf("provider '%s' requires a host\n"+
				"Use: nix-auth login <host> --provider %s", input, input)
		}

		// Create provider with config
		cfg := provider.ProviderConfig{
			Host:     host,
			ClientID: loginClientID,
		}
		prov := reg.New(cfg)
		return prov, host, nil
	}

	// Input is a host
	return resolveProviderForHost(input, providerFlag, timeout)
}

// resolveProviderForHost handles the case where input is a host
func resolveProviderForHost(host, providerFlag string, timeout int) (provider.Provider, string, error) {
	if providerFlag == "auto" {
		// Auto-detect provider type
		fmt.Printf("Detecting provider type for %s by querying API...\n", host)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		prov, err := provider.Detect(ctx, host, loginClientID)
		if err != nil {
			return nil, "", fmt.Errorf("failed to detect provider for %s: %w\n"+
				"Try: nix-auth login %s --provider <github|gitlab|gitea|forgejo>",
				host, err, host)
		}

		fmt.Printf("Detected: %s\n\n", prov.Name())
		return prov, host, nil
	}

	// Use explicitly specified provider
	cfg := provider.ProviderConfig{
		Host:     host,
		ClientID: loginClientID,
	}
	prov, ok := provider.GetWithConfig(providerFlag, cfg)
	if !ok {
		available := strings.Join(provider.List(), ", ")
		return nil, "", fmt.Errorf("unknown provider '%s'. Available providers: %s", providerFlag, available)
	}

	return prov, host, nil
}
