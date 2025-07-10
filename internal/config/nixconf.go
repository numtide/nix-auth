// Package config provides functionality for managing Nix configuration files and access tokens.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// filePermissions is the permission mode for nix.conf files.
	filePermissions = 0600
	// dirPermissions is the permission mode for configuration directories.
	dirPermissions = 0755
	// tokenSeparator is the separator used in access-tokens format.
	tokenSeparator = "="
	// tokenParts is the expected number of parts when splitting a token entry.
	tokenParts = 2
)

// NixConfig manages the nix.conf file.
type NixConfig struct {
	path string
}

// New creates a new NixConfig instance
// If configPath is empty, it will try to determine the path using:
// 1. NIX_USER_CONF_FILES environment variable (first file in the list)
// 2. XDG_CONFIG_HOME/nix/nix.conf
// 3. ~/.config/nix/nix.conf (default).
func New(configPath string) (*NixConfig, error) {
	if configPath == "" {
		configPath = DefaultUserConfigPath()
	}

	// Expand ~ to home directory if present
	configPath = expandTilde(configPath)

	return &NixConfig{path: configPath}, nil
}

// DefaultUserConfigPath returns the default path for the user's nix.conf based on environment variables
// This is the same logic used by New() when no path is provided.
func DefaultUserConfigPath() string {
	// Check NIX_USER_CONF_FILES first (colon-separated list)
	if nixUserConfFiles := os.Getenv("NIX_USER_CONF_FILES"); nixUserConfFiles != "" {
		// Use the first file in the list
		files := strings.Split(nixUserConfFiles, ":")
		if len(files) > 0 && files[0] != "" {
			return files[0]
		}
	}

	// Check XDG_CONFIG_HOME
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "nix", "nix.conf")
	}

	// Default to ~/.config/nix/nix.conf
	return "~/.config/nix/nix.conf"
}

// GetPath returns the config file path being used.
func (n *NixConfig) GetPath() string {
	return n.path
}

// GetToken retrieves the access token for a given host.
func (n *NixConfig) GetToken(host string) (string, error) {
	tokens, err := n.readAccessTokens()
	if err != nil {
		return "", err
	}

	return tokens[host], nil
}

// SetToken sets or updates the access token for a given host.
func (n *NixConfig) SetToken(host, token string) error {
	// Ensure directory exists
	dir := filepath.Dir(n.path)
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config
	lines, err := n.readConfigLines()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Create backup if file exists
	if _, err := os.Stat(n.path); err == nil {
		backupPath := fmt.Sprintf("%s.backup-%s", n.path, time.Now().Format("20060102-150405"))
		if err := n.createBackup(backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		fmt.Printf("Created backup: %s\n", backupPath)
	}

	// Update or add access-tokens line
	updated := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "access-tokens") {
			tokens, err := n.parseAccessTokensLine(line)
			if err != nil {
				return err
			}

			tokens[host] = token
			lines[i] = n.formatAccessTokensLine(tokens)
			updated = true

			break
		}
	}

	if !updated {
		// Add new access-tokens line
		tokens := map[string]string{host: token}
		lines = append(lines, n.formatAccessTokensLine(tokens))
	}

	// Write updated config
	return n.writeConfigLines(lines)
}

// RemoveToken removes the access token for a given host.
func (n *NixConfig) RemoveToken(host string) error {
	lines, err := n.readConfigLines()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no configuration file found")
		}

		return err
	}

	// Create backup
	backupPath := fmt.Sprintf("%s.backup-%s", n.path, time.Now().Format("20060102-150405"))
	if err := n.createBackup(backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Printf("Created backup: %s\n", backupPath)

	// Update access-tokens line
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "access-tokens") {
			continue
		}

		tokens, err := n.parseAccessTokensLine(line)
		if err != nil {
			return err
		}

		if _, ok := tokens[host]; !ok {
			continue
		}

		// Found the token to remove
		delete(tokens, host)

		found = true

		if len(tokens) == 0 {
			// Remove the line entirely if no tokens left
			lines = append(lines[:i], lines[i+1:]...)
		} else {
			lines[i] = n.formatAccessTokensLine(tokens)
		}

		break
	}

	if !found {
		return fmt.Errorf("no token found for %s", host)
	}

	return n.writeConfigLines(lines)
}

// ListTokens returns all configured access tokens (hosts only, not the actual tokens).
func (n *NixConfig) ListTokens() ([]string, error) {
	tokens, err := n.readAccessTokens()
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, err
	}

	hosts := make([]string, 0, len(tokens))
	for host := range tokens {
		hosts = append(hosts, host)
	}

	sort.Strings(hosts)

	return hosts, nil
}

func (n *NixConfig) readConfigLines() ([]string, error) {
	file, err := os.Open(n.path)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	var lines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func (n *NixConfig) writeConfigLines(lines []string) error {
	file, err := os.OpenFile(n.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	return writer.Flush()
}

func (n *NixConfig) readAccessTokens() (map[string]string, error) {
	lines, err := n.readConfigLines()
	if err != nil {
		return nil, err
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "access-tokens") {
			return n.parseAccessTokensLine(line)
		}
	}

	return map[string]string{}, nil
}

func (n *NixConfig) parseAccessTokensLine(line string) (map[string]string, error) {
	tokens := make(map[string]string)

	// Find the position of "access-tokens"
	idx := strings.Index(line, "access-tokens")
	if idx == -1 {
		return nil, fmt.Errorf("invalid access-tokens line")
	}

	// Find the equals sign after "access-tokens"
	eqIdx := strings.Index(line[idx:], "=")
	if eqIdx == -1 {
		return nil, fmt.Errorf("invalid access-tokens line format")
	}

	eqIdx += idx

	// Get the token pairs after the equals sign
	tokenPart := strings.TrimSpace(line[eqIdx+1:])
	if tokenPart == "" {
		return tokens, nil
	}

	tokenPairs := strings.Fields(tokenPart)
	for _, pair := range tokenPairs {
		hostToken := strings.SplitN(pair, tokenSeparator, tokenParts)
		if len(hostToken) != tokenParts {
			return nil, fmt.Errorf("invalid token format: %s", pair)
		}

		tokens[hostToken[0]] = hostToken[1]
	}

	return tokens, nil
}

func (n *NixConfig) formatAccessTokensLine(tokens map[string]string) string {
	// Sort hosts for deterministic output
	hosts := make([]string, 0, len(tokens))
	for host := range tokens {
		hosts = append(hosts, host)
	}

	sort.Strings(hosts)

	pairs := make([]string, 0, len(hosts))
	for _, host := range hosts {
		pairs = append(pairs, fmt.Sprintf("%s=%s", host, tokens[host]))
	}

	return fmt.Sprintf("access-tokens = %s", strings.Join(pairs, " "))
}

func (n *NixConfig) createBackup(backupPath string) error {
	input, err := os.ReadFile(n.path)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, input, filePermissions)
}

// expandTilde expands ~ to the user's home directory.
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}

		return filepath.Join(homeDir, path[2:])
	}

	return path
}
