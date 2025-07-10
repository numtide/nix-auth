package ui

import (
	"strings"
	"testing"
)

// hasKnownTokenPrefix checks if a token has a known prefix.
func hasKnownTokenPrefix(token string) bool {
	knownPrefixes := []string{"gho_", "ghp_", "ghs_", "github_pat_", "glpat-", "gloas-", "glrt-", "gitea_"}
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(token, prefix) {
			return true
		}
	}

	return false
}

// checkTokenSecurityExposure validates that masked tokens don't expose sensitive parts.
func checkTokenSecurityExposure(t *testing.T, token, result string) {
	t.Helper()

	if len(token) < 14 {
		return
	}

	hasKnownPrefix := hasKnownTokenPrefix(token)

	if hasKnownPrefix && len(token) > 8 {
		// Check we're not exposing more than last 2 chars
		suffix := token[len(token)-8 : len(token)-2]
		if strings.Contains(result, suffix) {
			t.Errorf("MaskToken exposed too much of token suffix: %q contains %q", result, suffix)
		}
	} else if !hasKnownPrefix {
		// For unknown tokens, check we're not exposing any suffix
		suffix := token[len(token)-8:]
		if strings.Contains(result, suffix) {
			t.Errorf("MaskToken exposed token suffix: %q contains %q", result, suffix)
		}
	}

	// Check that we're not exposing too much of the middle
	if len(token) > 20 {
		middle := token[8 : len(token)-8]
		for i := 0; i < len(middle)-3; i++ {
			if strings.Contains(result, middle[i:i+4]) {
				t.Errorf("MaskToken exposed middle section of token: %q contains %q", result, middle[i:i+4])
			}
		}
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		// Empty and short tokens
		{
			name:     "empty token",
			token:    "",
			expected: "********",
		},
		{
			name:     "very short token",
			token:    "abc",
			expected: "********",
		},
		{
			name:     "token exactly 15 chars",
			token:    "123456789012345",
			expected: "1234********",
		},
		{
			name:     "token exactly 16 chars",
			token:    "1234567890123456",
			expected: "1234********",
		},

		// GitHub tokens
		{
			name:     "GitHub OAuth token",
			token:    "gho_16C7e42F292c6912E7710c838347Ae178B4a",
			expected: "gho_******4a",
		},
		{
			name:     "GitHub personal access token",
			token:    "ghp_16C7e42F292c6912E7710c838347Ae178B4a",
			expected: "ghp_******4a",
		},
		{
			name:     "GitHub server token",
			token:    "ghs_16C7e42F292c6912E7710c838347Ae178B4a",
			expected: "ghs_******4a",
		},
		{
			name:     "GitHub fine-grained PAT",
			token:    "github_pat_11ABCDEF0_1234567890abcdef",
			expected: "github_pat_******ef",
		},

		// GitLab tokens
		{
			name:     "GitLab personal access token",
			token:    "glpat-1234567890abcdefghij",
			expected: "glpat-******ij",
		},
		{
			name:     "GitLab OAuth access token",
			token:    "gloas-1234567890abcdefghij",
			expected: "gloas-******ij",
		},
		{
			name:     "GitLab refresh token",
			token:    "glrt-1234567890abcdefghij",
			expected: "glrt-******ij",
		},

		// Gitea tokens
		{
			name:     "Gitea token",
			token:    "gitea_1234567890abcdefghij",
			expected: "gitea_******ij",
		},

		// Unknown token types
		{
			name:     "unknown token type",
			token:    "xyz_1234567890abcdefghij",
			expected: "xyz_********",
		},
		{
			name:     "random long token",
			token:    "abcdefghijklmnopqrstuvwxyz1234567890",
			expected: "abcd********",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskToken(tt.token)
			if result != tt.expected {
				t.Errorf("MaskToken(%q) = %q, want %q", tt.token, result, tt.expected)
			}

			// Security check: ensure no sensitive part of the token is exposed
			checkTokenSecurityExposure(t, tt.token, result)
		})
	}
}

func TestMaskTokenSecurity(t *testing.T) {
	// Test that the function handles Unicode correctly
	t.Run("unicode token", func(t *testing.T) {
		token := "test_こんにちは世界1234567890" //nolint:gosec // test token
		result := MaskToken(token)
		// Should show first 4 bytes, not break Unicode
		if result != "test********" {
			t.Errorf("MaskToken failed to handle Unicode correctly: got %q", result)
		}
	})

	// Test consistent masking length
	t.Run("consistent mask length", func(t *testing.T) {
		tokens := []struct {
			token    string
			expected string
		}{
			{"gho_shorttoken123", "gho_******23"},
			{"gho_verylongtokenwithmanymorecharacters123456789", "gho_******89"},
		}
		for _, tt := range tokens {
			result := MaskToken(tt.token)
			if result != tt.expected {
				t.Errorf("MaskToken inconsistent masking for %q: got %q, want %q", tt.token, result, tt.expected)
			}
		}
	})
}
