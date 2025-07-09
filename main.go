package main

import (
	"os"

	"github.com/numtide/nix-auth/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
