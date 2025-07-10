package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/numtide/nix-auth/internal/config"
	"github.com/numtide/nix-auth/internal/provider"
	"github.com/numtide/nix-auth/internal/util"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	setTokenForce    bool
	setTokenProvider string
)

var setTokenCmd = &cobra.Command{
	Use:   "set-token <host> [token]",
	Short: "Set an access token for a specific host",
	Long: `Set an access token for a specific host.

The token can be provided as an argument or entered interactively for security.
If a provider is specified or detected, the token will be validated before saving.`,
	Example: `  # Set token directly
  nix-auth set-token github.com ghp_xxxxxxxxxxxx

  # Prompt for token (more secure)
  nix-auth set-token github.com

  # Force replace existing token
  nix-auth set-token github.com ghp_xxxxxxxxxxxx --force

  # Specify provider for validation
  nix-auth set-token git.company.com --provider gitlab`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		host := args[0]

		// Initialize config
		cfg, err := config.New(configPath)
		if err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
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
				maskedExisting := util.MaskToken(existingToken)
				fmt.Printf("Token already exists for %s: %s\n", host, maskedExisting)
				fmt.Print("Replace it? (y/N): ")

				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Operation cancelled")
					return nil
				}
			}
		}

		// Get token from args or prompt
		var token string
		if len(args) == 2 {
			token = args[1]
		} else {
			fmt.Printf("Enter token for %s: ", host)
			
			// Check if stdin is a terminal
			if term.IsTerminal(int(syscall.Stdin)) {
				// Use secure password input for terminals
				byteToken, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println() // Add newline after password input
				if err != nil {
					return fmt.Errorf("failed to read token: %w", err)
				}
				token = string(byteToken)
			} else {
				// For non-terminal input (like tests or piped input)
				reader := bufio.NewReader(os.Stdin)
				tokenBytes, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read token: %w", err)
				}
				token = strings.TrimSuffix(tokenBytes, "\n")
			}
		}

		// Trim whitespace
		token = strings.TrimSpace(token)
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

		maskedToken := util.MaskToken(token)
		fmt.Printf("Successfully set token for %s: %s\n", host, maskedToken)
		fmt.Printf("Config saved to: %s\n", cfg.GetPath())

		return nil
	},
}

func init() {
	setTokenCmd.Flags().BoolVarP(&setTokenForce, "force", "f", false, "Force replace existing token without confirmation")
	setTokenCmd.Flags().StringVarP(&setTokenProvider, "provider", "p", "", "Specify provider for token validation (e.g., github, gitlab)")
}
