package nixconf

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Parser parses nix config files while preserving formatting and comments.
type Parser struct {
	visited map[string]bool
}

// ConfigLine represents a single line in the config with metadata.
type ConfigLine struct {
	Raw         string // Original line with all whitespace and comments
	Key         string // Setting name (empty for non-setting lines)
	Value       string // Setting value (empty for non-setting lines)
	IsInclude   bool   // True if this is an include directive
	IncludePath string // Path for include directive
	SourceFile  string // Which file this line came from
	LineNum     int    // Line number in source file
}

// ParsedConfig preserves original formatting while tracking settings.
type ParsedConfig struct {
	Lines    []ConfigLine
	Settings map[string]string // For quick lookup
	Includes map[string]bool   // Track which includes are present
}

// NewParsedConfig creates a new empty ParsedConfig.
func NewParsedConfig() *ParsedConfig {
	return &ParsedConfig{
		Lines:    []ConfigLine{},
		Settings: make(map[string]string),
		Includes: make(map[string]bool),
	}
}

// NewParser creates a parser that preserves formatting.
func NewParser() *Parser {
	return &Parser{
		visited: make(map[string]bool),
	}
}

// ParseFile parses a config file preserving all formatting.
func (p *Parser) ParseFile(path string) (*ParsedConfig, error) {
	config := NewParsedConfig()

	p.visited = make(map[string]bool)
	if err := p.parseFileRecursive(path, config); err != nil {
		return nil, err
	}

	return config, nil
}

func (p *Parser) parseFileRecursive(path string, config *ParsedConfig) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", path, err)
	}

	if p.visited[absPath] {
		return fmt.Errorf("circular include detected: %s", absPath)
	}

	p.visited[absPath] = true

	file, err := os.Open(absPath) //nolint:gosec // trusted config file path
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		rawLine := scanner.Text()

		line := ConfigLine{
			Raw:        rawLine,
			SourceFile: absPath,
			LineNum:    lineNum,
		}

		// Parse the line without modifying it
		p.parseLine(&line)

		// Handle includes and settings
		if line.IsInclude {
			if err := p.handleInclude(&line, rawLine, absPath, lineNum, config); err != nil {
				return err
			}
		} else if line.Key != "" {
			// Regular setting - track it
			config.Settings[line.Key] = line.Value
		}

		// Always preserve the line
		config.Lines = append(config.Lines, line)
	}

	return scanner.Err()
}

// handleInclude processes an include directive.
func (p *Parser) handleInclude(line *ConfigLine, rawLine, absPath string, lineNum int, config *ParsedConfig) error {
	includePath := line.IncludePath
	if !filepath.IsAbs(includePath) {
		includePath = filepath.Join(filepath.Dir(absPath), includePath)
	}

	// Track that we have this include
	config.Includes[line.IncludePath] = true

	// Try to parse included file
	err := p.parseFileRecursive(includePath, config)
	if err != nil {
		// !include ignores missing files
		if !strings.HasPrefix(strings.TrimSpace(rawLine), "!include") || !os.IsNotExist(err) {
			return fmt.Errorf("failed to include %s from %s:%d: %w", includePath, absPath, lineNum, err)
		}
	}

	return nil
}

// parseLine extracts key/value from a line without modifying it.
func (p *Parser) parseLine(line *ConfigLine) {
	// Find content before any comment
	content := line.Raw
	if idx := strings.IndexByte(content, '#'); idx != -1 {
		content = content[:idx]
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return
	}

	// Check for include directive
	if strings.HasPrefix(trimmed, "include ") || strings.HasPrefix(trimmed, "!include ") {
		line.IsInclude = true
		parts := strings.Fields(trimmed)

		const minPartsForInclude = 2
		if len(parts) >= minPartsForInclude {
			line.IncludePath = parts[1]
		}

		return
	}

	// Check for setting (key = value)
	if idx := strings.Index(trimmed, "="); idx != -1 {
		key := strings.TrimSpace(trimmed[:idx])
		value := strings.TrimSpace(trimmed[idx+1:])
		line.Key = key
		line.Value = value
	}
}

// FindSettingLine finds the line containing a specific setting.
func (c *ParsedConfig) FindSettingLine(key string) *ConfigLine {
	// Search backwards to find the last occurrence
	for i := len(c.Lines) - 1; i >= 0; i-- {
		if c.Lines[i].Key == key {
			return &c.Lines[i]
		}
	}

	return nil
}

// HasInclude checks if an include directive exists.
func (c *ParsedConfig) HasInclude(path string) bool {
	return c.Includes[path]
}

// WriteToFile writes the config back preserving all formatting.
func (c *ParsedConfig) WriteToFile(path string, lines []ConfigLine) error {
	// Get original file permissions if it exists
	const defaultPerms = 0o644

	var perms os.FileMode = defaultPerms
	if info, err := os.Stat(path); err == nil {
		perms = info.Mode()
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms) //nolint:gosec // trusted config file path
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line.Raw + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// ParseAccessTokens parses the access-tokens setting value into a map.
func ParseAccessTokens(value string) (map[string]string, error) {
	tokens := make(map[string]string)

	if value == "" {
		return tokens, nil
	}

	// Split on whitespace
	pairs := strings.Fields(value)

	for _, pair := range pairs {
		// Split on first =
		const expectedParts = 2

		parts := strings.SplitN(pair, "=", expectedParts)
		if len(parts) != expectedParts {
			return nil, fmt.Errorf("invalid token format: %s", pair)
		}

		host := parts[0]
		token := parts[1]

		if host == "" || token == "" {
			return nil, fmt.Errorf("invalid token format: empty host or token in %s", pair)
		}

		tokens[host] = token
	}

	return tokens, nil
}

// FormatAccessTokens formats a token map into the access-tokens value format.
func FormatAccessTokens(tokens map[string]string) string {
	if len(tokens) == 0 {
		return ""
	}

	// Sort hosts for deterministic output
	hosts := make([]string, 0, len(tokens))
	for host := range tokens {
		hosts = append(hosts, host)
	}

	sort.Strings(hosts)

	// Build sorted pairs
	pairs := make([]string, 0, len(tokens))
	for _, host := range hosts {
		pairs = append(pairs, fmt.Sprintf("%s=%s", host, tokens[host]))
	}

	return strings.Join(pairs, " ")
}
