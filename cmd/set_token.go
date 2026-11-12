package cmd

import (
	"context"
	"fmt"

	"github.com/numtide/nix-auth/internal/nixconf"
	"github.com/numtide/nix-auth/internal/provider"
	"github.com/numtide/nix-auth/internal/ui"
	"github.com/spf13/cobra"
)

const (
	minSetTokenArgs = 1
	maxSetTokenArgs = 2
)

var (
	setTokenForce    bool
	setTokenProvider string
	setTokenNetrc    bool
	setTokenUser     string
)

var setTokenCmd = &cobra.Command{
	Use:   "set-token <host> [token]",
	Short: "Set an access token for a specific host",
	Long: `Set an access token for a specific host.

The token can be provided as an argument or entered interactively for security.
If a provider is specified or detected, the token will be validated before saving.

With --netrc the credentials are also written to the netrc file next to nix.conf.
With --user the host is treated as a Basic Auth service (e.g. a private binary cache):
the credentials are written only to the netrc file, with no provider validation.`,
	Example: `  # Set token directly
  nix-auth set-token github.com ghp_xxxxxxxxxxxx

  # Prompt for token (more secure)
  nix-auth set-token github.com

  # Force replace existing token
  nix-auth set-token github.com ghp_xxxxxxxxxxxx --force

  # Specify provider for validation
  nix-auth set-token git.company.com --provider gitlab

  # Also write to netrc
  nix-auth set-token github.com --netrc

  # Basic Auth binary cache (netrc only)
  nix-auth set-token cache.example.com --user alice`,
	Args: cobra.RangeArgs(minSetTokenArgs, maxSetTokenArgs),
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := context.Background()
		host := args[0]

		// Initialize config
		cfg, err := nixconf.New(configPath)
		if err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		if setTokenUser != "" {
			return setBasicAuth(cfg, host, args)
		}

		// Check if token already exists
		hosts, err := cfg.ListTokens()
		if err != nil {
			return fmt.Errorf("failed to list tokens: %w", err)
		}

		tokenExists := false
		for _, h := range hosts {
			if h == host {
				tokenExists = true
				break
			}
		}

		if tokenExists && !setTokenForce {
			existingToken, err := cfg.GetToken(host)
			if err == nil && existingToken != "" {
				maskedExisting := ui.MaskToken(existingToken)
				fmt.Printf("Token already exists for %s: %s\n", host, maskedExisting)

				confirm, err := ui.ReadYesNo("Replace it? (y/N): ")
				if err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}
				if !confirm {
					fmt.Println("Operation cancelled")
					return nil
				}
			}
		}

		// Get token from args or prompt
		var token string
		if len(args) == maxSetTokenArgs {
			token = args[1]
		} else {
			var err error
			token, err = ui.ReadSecureInput(fmt.Sprintf("Enter token for %s: ", host))
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
		}

		// Check if token is empty
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		// Determine provider
		if setTokenProvider != "" {
			// User specified provider
			p, ok := provider.Get(setTokenProvider)
			if !ok {
				return fmt.Errorf("unknown provider: %s", setTokenProvider)
			}
			// Validate token if provider is available
			fmt.Printf("Validating token with %s provider...\n", p.Name())
			status, err := p.ValidateToken(ctx, token)
			if err != nil {
				return fmt.Errorf("token validation failed: %w", err)
			}
			if status != provider.ValidationStatusValid {
				return fmt.Errorf("token is not valid")
			}
			fmt.Println("Token validated successfully")
		} else {
			// Try to detect provider from host
			p, err := provider.Detect(ctx, host, "")
			if err == nil && p.Name() != "unknown" {
				// Validate token if provider was detected
				fmt.Printf("Detected %s provider, validating token...\n", p.Name())
				status, err := p.ValidateToken(ctx, token)
				if err != nil {
					// Just warn, don't fail
					fmt.Printf("Warning: token validation failed: %v\n", err)
				} else if status != provider.ValidationStatusValid {
					fmt.Printf("Warning: token may not be valid\n")
				} else {
					fmt.Println("Token validated successfully")
				}
			}
		}

		// Set the token
		if err := cfg.SetToken(host, token); err != nil {
			return fmt.Errorf("failed to set token: %w", err)
		}

		maskedToken := ui.MaskToken(token)
		fmt.Printf("Successfully set token for %s: %s\n", host, maskedToken)
		fmt.Printf("Config saved to: %s\n", cfg.GetTokenFilePath())

		if setTokenNetrc {
			return saveNetrc(cfg, host, defaultNetrcLogin, token)
		}

		return nil
	},
}

func init() {
	setTokenCmd.Flags().BoolVarP(&setTokenForce, "force", "f", false, "Force replace existing token without confirmation")
	setTokenCmd.Flags().BoolVar(&setTokenNetrc, "netrc", false, "Also write the token to the netrc file")
	setTokenCmd.Flags().StringVar(&setTokenUser, "user", "", "Basic Auth username; writes only to the netrc file")
	setTokenCmd.Flags().StringVarP(&setTokenProvider, "provider", "p", "", "Specify provider for token validation (e.g., github, gitlab)")
}

// setBasicAuth stores user/password for a Basic Auth host in the netrc file only.
func setBasicAuth(cfg *nixconf.NixConfig, host string, args []string) error {
	var password string

	if len(args) == maxSetTokenArgs {
		password = args[1]
	} else {
		var err error

		password, err = ui.ReadSecureInput(fmt.Sprintf("Enter password for %s@%s: ", setTokenUser, host))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if err := saveNetrc(cfg, host, setTokenUser, password); err != nil {
		return err
	}

	fmt.Printf("Successfully set credentials for %s (user %s)\n", host, setTokenUser)

	return nil
}
