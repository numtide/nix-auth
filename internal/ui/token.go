package ui

import (
	"fmt"
	"strings"
)

const (
	// minTokenLength is the minimum length for a token to show partial info.
	minTokenLength = 14
	// defaultMaskLength is the default mask length for tokens.
	defaultMaskLength = 8
	// middleMaskLength is the mask length for middle section of known tokens.
	middleMaskLength = 6
	// prefixLength is the length of non-prefix token display.
	prefixLength = 4
	// suffixLength is the length of suffix to show for known tokens.
	suffixLength = 2
)

// MaskToken masks a token for security, showing only the token prefix for known types.
func MaskToken(token string) string {
	// Handle empty or very short tokens
	if len(token) < minTokenLength {
		return strings.Repeat("*", defaultMaskLength)
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
			if len(token) >= len(prefix)+defaultMaskLength {
				return fmt.Sprintf("%s%s%s", prefix, strings.Repeat("*", middleMaskLength), token[len(token)-suffixLength:])
			}
			// Fallback if token is too short
			return fmt.Sprintf("%s%s", prefix, strings.Repeat("*", defaultMaskLength))
		}
	}

	// For unknown token types, show first 4 chars (might indicate type) + mask
	// This is more conservative than showing both prefix and suffix
	return fmt.Sprintf("%s%s", token[:prefixLength], strings.Repeat("*", defaultMaskLength))
}
