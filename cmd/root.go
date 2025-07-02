package cmd

import (
	"fmt"
	"github.com/numtide/nix-auth/internal/config"
	"github.com/spf13/cobra"
)

var (
	configPath string
	rootCmd    = &cobra.Command{
		Use:   "nix-auth",
		Short: "Manage access tokens for Nix flakes",
		Long: `nix-auth is a CLI tool that helps you configure access tokens
for various Git providers (GitHub, GitLab, etc.) to avoid rate limits when
using Nix flakes.`,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add persistent flag for config path
	defaultPath := config.DefaultUserConfigPath()
	flagDesc := fmt.Sprintf("Path to nix.conf file (default: %s)", defaultPath)
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", flagDesc)

	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logoutCmd)
}
