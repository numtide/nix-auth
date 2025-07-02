package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NixConfig manages the nix.conf file
type NixConfig struct {
	path string
}

// New creates a new NixConfig instance
func New() (*NixConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "nix", "nix.conf")
	return &NixConfig{path: configPath}, nil
}

// GetToken retrieves the access token for a given host
func (n *NixConfig) GetToken(host string) (string, error) {
	tokens, err := n.readAccessTokens()
	if err != nil {
		return "", err
	}

	return tokens[host], nil
}

// SetToken sets or updates the access token for a given host
func (n *NixConfig) SetToken(host, token string) error {
	// Ensure directory exists
	dir := filepath.Dir(n.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
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

// RemoveToken removes the access token for a given host
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
		if strings.HasPrefix(trimmed, "access-tokens") {
			tokens, err := n.parseAccessTokensLine(line)
			if err != nil {
				return err
			}
			if _, ok := tokens[host]; ok {
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
		}
	}

	if !found {
		return fmt.Errorf("no token found for %s", host)
	}

	return n.writeConfigLines(lines)
}

// ListTokens returns all configured access tokens (hosts only, not the actual tokens)
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
	return hosts, nil
}

func (n *NixConfig) readConfigLines() ([]string, error) {
	file, err := os.Open(n.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func (n *NixConfig) writeConfigLines(lines []string) error {
	file, err := os.OpenFile(n.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

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

	// Remove "access-tokens = " prefix
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid access-tokens line format")
	}

	tokenPairs := strings.Fields(parts[1])
	for _, pair := range tokenPairs {
		hostToken := strings.SplitN(pair, "=", 2)
		if len(hostToken) != 2 {
			return nil, fmt.Errorf("invalid token format: %s", pair)
		}
		tokens[hostToken[0]] = hostToken[1]
	}

	return tokens, nil
}

func (n *NixConfig) formatAccessTokensLine(tokens map[string]string) string {
	var pairs []string
	for host, token := range tokens {
		pairs = append(pairs, fmt.Sprintf("%s=%s", host, token))
	}
	return fmt.Sprintf("access-tokens = %s", strings.Join(pairs, " "))
}

func (n *NixConfig) createBackup(backupPath string) error {
	input, err := os.ReadFile(n.path)
	if err != nil {
		return err
	}
	return os.WriteFile(backupPath, input, 0644)
}
