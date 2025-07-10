// Package version provides version information for nix-auth.
package version

import "fmt"

var (
	// Version is the main version number that is being run at the moment.
	Version = "dev"

	// Commit is the git commit that was compiled. This will be filled in by the compiler.
	Commit = "none"

	// Date is the build date in RFC3339 format.
	Date = "unknown"
)

// String returns the complete version string.
func String() string {
	if Version == "dev" {
		return fmt.Sprintf("%s (commit: %s, built at: %s)", Version, Commit, Date)
	}

	return Version
}

// Full returns the full version information.
func Full() string {
	return fmt.Sprintf("nix-auth %s\ncommit: %s\nbuilt at: %s", Version, Commit, Date)
}
