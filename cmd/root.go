package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nix-auth",
	Short: "Manage access tokens for Nix flakes",
	Long: `nix-auth is a CLI tool that helps you configure access tokens
for various Git providers (GitHub, GitLab, etc.) to avoid rate limits when
using Nix flakes.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logoutCmd)
}
