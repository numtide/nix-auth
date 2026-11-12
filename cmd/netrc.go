package cmd

import (
	"errors"
	"fmt"

	"github.com/numtide/nix-auth/internal/netrc"
	"github.com/numtide/nix-auth/internal/nixconf"
)

// defaultNetrcLogin is the login written for OAuth/PAT tokens, which
// providers accept as the Basic Auth username alongside the token.
const defaultNetrcLogin = "oauth2"

// saveNetrc writes host credentials to the netrc file managed next to nix.conf
// and makes sure nix.conf points Nix at it.
func saveNetrc(cfg *nixconf.NixConfig, host, login, password string) error {
	file := netrc.New(cfg.GetNetrcFilePath())

	if err := file.Set(netrc.Entry{Machine: host, Login: login, Password: password}); err != nil {
		return fmt.Errorf("failed to write netrc: %w", err)
	}

	fmt.Printf("Netrc entry saved to: %s\n", file.Path())

	changed, current, err := cfg.EnsureNetrcFile()

	switch {
	case errors.Is(err, nixconf.ErrNetrcFileConflict):
		fmt.Printf("Warning: nix.conf already sets netrc-file = %s; not overriding it.\n"+
			"Nix will not read %s unless you point netrc-file at it.\n", current, file.Path())
	case err != nil:
		return err
	case changed:
		fmt.Printf("Added netrc-file = %s to %s\n", current, cfg.GetPath())
	}

	return nil
}

// removeNetrc deletes the host entry from the managed netrc file.
// It reports whether an entry was removed.
func removeNetrc(cfg *nixconf.NixConfig, host string) (bool, error) {
	return netrc.New(cfg.GetNetrcFilePath()).Remove(host)
}
