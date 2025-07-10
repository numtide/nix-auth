package cmd

import (
	"fmt"

	"github.com/numtide/nix-auth/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println(version.Full())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
