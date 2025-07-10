package ui

import (
	"fmt"
	"strings"
)

// MaskToken masks a token for security, showing only the token prefix for known types
func MaskToken(token string) string {
	// Handle empty or very short tokens
	if len(token) < 14 {
		return strings.Repeat("*", 8)
	}

	// Known token prefixes - these help identify the token type without revealing sensitive data
	knownPrefixes := []string{
		"gho_",        // GitHub OAuth token
		"ghp_",        // GitHub personal access token
		"ghs_",        // GitHub server-to-server token
		"github_pat_", // GitHub fine-grained PAT
		"glpat-",      // GitLab personal access token
		"gloas-",      // GitLab OAuth access token
		"glrt-",       // GitLab refresh token
		"gitea_",      // Gitea token prefix (if standardized)
	}

	// Check if token starts with a known prefix
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(token, prefix) {
			// Show prefix + last 2 chars for better differentiation between multiple tokens
			if len(token) >= len(prefix)+8 {
				return fmt.Sprintf("%s%s%s", prefix, strings.Repeat("*", 6), token[len(token)-2:])
			}
			// Fallback if token is too short
			return fmt.Sprintf("%s%s", prefix, strings.Repeat("*", 8))
		}
	}

	// For unknown token types, show first 4 chars (might indicate type) + mask
	// This is more conservative than showing both prefix and suffix
	return fmt.Sprintf("%s%s", token[:4], strings.Repeat("*", 8))
}
