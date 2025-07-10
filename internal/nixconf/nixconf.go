// Package nixconf manages Nix configuration files and access tokens.
package nixconf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// accessTokensFile is the default name for the separate tokens file.
	accessTokensFile = "access-tokens.conf"
	// tokenFilePermissions is the permission mode for the tokens file.
	tokenFilePermissions = 0600
	// dirPermissions is the permission mode for configuration directories.
	dirPermissions = 0755
	// backupTimeFormat is the time format used for backup file names.
	backupTimeFormat = "20060102-150405"
	// accessTokensKey is the config key for access tokens.
	accessTokensKey = "access-tokens"
)

// NixConfig manages the nix.conf file with minimal modifications.
type NixConfig struct {
	mainPath string
	parser   *Parser
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

	configPath = expandTilde(configPath)

	return &NixConfig{
		mainPath: configPath,
		parser:   NewParser(),
	}, nil
}

// DefaultUserConfigPath returns the default path for the user's nix.conf based on environment variables.
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
	return n.mainPath
}

// GetToken retrieves the access token for a given host.
func (n *NixConfig) GetToken(host string) (string, error) {
	config, err := n.parser.ParseFile(n.mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", err
	}

	if tokenValue, exists := config.Settings[accessTokensKey]; exists {
		tokens, err := ParseAccessTokens(tokenValue)
		if err != nil {
			return "", err
		}

		return tokens[host], nil
	}

	return "", nil
}

// SetToken sets or updates the access token for a given host.
func (n *NixConfig) SetToken(host, token string) error {
	// Ensure directory exists
	dir := filepath.Dir(n.mainPath)
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Parse existing configuration
	config, err := n.parser.ParseFile(n.mainPath)
	mainFileExists := err == nil

	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to parse config: %w", err)
		}
		// File doesn't exist - create empty config
		config = NewParsedConfig()
	}

	// Get all existing tokens
	existingTokens := make(map[string]string)

	if tokenValue, exists := config.Settings[accessTokensKey]; exists {
		var err error
		existingTokens, err = ParseAccessTokens(tokenValue)

		if err != nil {
			return fmt.Errorf("failed to parse existing tokens: %w", err)
		}
	}

	// Add/update the token
	existingTokens[host] = token

	// Check if tokens are in main config file
	tokenLine := config.FindSettingLine(accessTokensKey)
	tokensInMainFile := tokenLine != nil && strings.HasSuffix(tokenLine.SourceFile, filepath.Base(n.mainPath))

	// First, write all tokens to the token file
	tokenFilePath := n.GetTokenFilePath()
	if err := n.writeTokenFile(tokenFilePath, existingTokens); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	// Then update main config if needed
	if !mainFileExists {
		// New config file - create with include
		lines := []ConfigLine{
			{Raw: "# Nix configuration", SourceFile: n.mainPath},
			{Raw: "!include " + accessTokensFile, SourceFile: n.mainPath},
		}
		if err := config.WriteToFile(n.mainPath, lines); err != nil {
			return fmt.Errorf("failed to create main config: %w", err)
		}
	} else if tokensInMainFile || !config.HasInclude(accessTokensFile) {
		if tokensInMainFile {
			tokenFilePath := n.GetTokenFilePath()
			fmt.Printf("Migrating tokens to secure file: %s\n", tokenFilePath)
		}

		// Need to update existing file: either migrate tokens or add missing include
		if err := n.updateMainConfig(config); err != nil {
			return err
		}
	}

	return nil
}

// updateMainConfig updates the main config to include the token file and remove any access-tokens.
func (n *NixConfig) updateMainConfig(config *ParsedConfig) error {
	// Create backup
	backupPath := fmt.Sprintf("%s.backup-%s", n.mainPath, time.Now().Format(backupTimeFormat))
	if err := n.createBackup(n.mainPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Printf("Created backup: %s\n", backupPath)

	// Replace access-tokens line with include directive (or just add include if no tokens)
	newLines := n.replaceTokensWithInclude(config)

	// Write updated main config
	if err := config.WriteToFile(n.mainPath, newLines); err != nil {
		return fmt.Errorf("failed to update main config: %w", err)
	}

	return nil
}

// replaceTokensWithInclude replaces access-tokens lines with include directive, or appends it if no tokens found.
func (n *NixConfig) replaceTokensWithInclude(config *ParsedConfig) []ConfigLine {
	newLines := make([]ConfigLine, 0, len(config.Lines))
	tokenLineFound := false

	for _, line := range config.Lines {
		// Replace access-tokens line with include directive
		if line.Key == accessTokensKey && strings.HasSuffix(line.SourceFile, filepath.Base(n.mainPath)) {
			// Replace this line with include directive
			includeLine := ConfigLine{
				Raw:        "!include " + accessTokensFile,
				SourceFile: n.mainPath,
			}
			newLines = append(newLines, includeLine)
			tokenLineFound = true

			continue
		}

		newLines = append(newLines, line)
	}

	// If no token line was found, append include at the end
	if !tokenLineFound {
		includeLine := ConfigLine{
			Raw:        "!include " + accessTokensFile,
			SourceFile: n.mainPath,
		}
		newLines = append(newLines, includeLine)
	}

	return newLines
}

// RemoveToken removes the access token for a given host.
func (n *NixConfig) RemoveToken(host string) error {
	config, err := n.parser.ParseFile(n.mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no configuration file found")
		}

		return err
	}

	tokenValue, exists := config.Settings[accessTokensKey]
	if !exists {
		return fmt.Errorf("no tokens configured")
	}

	tokens, err := ParseAccessTokens(tokenValue)
	if err != nil {
		return err
	}

	if _, exists := tokens[host]; !exists {
		return fmt.Errorf("no token found for %s", host)
	}

	// Remove the token
	delete(tokens, host)

	// Update token file
	tokenFilePath := n.GetTokenFilePath()
	if len(tokens) == 0 {
		// Remove token file if empty
		if err := os.Remove(tokenFilePath); err != nil && !os.IsNotExist(err) {
			return err
		}

		return nil
	}

	return n.writeTokenFile(tokenFilePath, tokens)
}

// ListTokens returns all configured access tokens (hosts only).
func (n *NixConfig) ListTokens() ([]string, error) {
	config, err := n.parser.ParseFile(n.mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, err
	}

	tokenValue, exists := config.Settings[accessTokensKey]
	if !exists {
		return []string{}, nil
	}

	tokens, err := ParseAccessTokens(tokenValue)
	if err != nil {
		return nil, err
	}

	hosts := make([]string, 0, len(tokens))
	for host := range tokens {
		hosts = append(hosts, host)
	}

	sort.Strings(hosts)

	return hosts, nil
}

// GetTokenFilePath returns the path to the token file.
func (n *NixConfig) GetTokenFilePath() string {
	return filepath.Join(filepath.Dir(n.mainPath), accessTokensFile)
}

// writeTokenFile writes tokens to the token file with restricted permissions.
func (n *NixConfig) writeTokenFile(path string, tokens map[string]string) error {
	content := FormatAccessTokens(tokens)
	if content != "" {
		content = accessTokensKey + " = " + content + "\n"
	}

	return os.WriteFile(path, []byte(content), tokenFilePermissions)
}

// createBackup creates a backup of a file preserving permissions.
func (n *NixConfig) createBackup(src, dst string) error {
	input, err := os.ReadFile(src) //nolint:gosec // trusted config file path
	if err != nil {
		return err
	}

	// Get original file permissions
	perms := os.FileMode(tokenFilePermissions)
	if info, err := os.Stat(src); err == nil {
		perms = info.Mode()
	}

	return os.WriteFile(dst, input, perms)
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
