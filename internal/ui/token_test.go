package ui

import (
	"strings"
	"testing"
)

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
			if len(tt.token) >= 14 {
				// For known prefixes, we show last 2 chars
				hasKnownPrefix := false
				for _, prefix := range []string{"gho_", "ghp_", "ghs_", "github_pat_", "glpat-", "gloas-", "glrt-", "gitea_"} {
					if strings.HasPrefix(tt.token, prefix) {
						hasKnownPrefix = true
						break
					}
				}

				if hasKnownPrefix && len(tt.token) > 8 {
					// Check we're not exposing more than last 2 chars
					suffix := tt.token[len(tt.token)-8 : len(tt.token)-2]
					if strings.Contains(result, suffix) {
						t.Errorf("MaskToken exposed too much of token suffix: %q contains %q", result, suffix)
					}
				} else if !hasKnownPrefix {
					// For unknown tokens, check we're not exposing any suffix
					suffix := tt.token[len(tt.token)-8:]
					if strings.Contains(result, suffix) {
						t.Errorf("MaskToken exposed token suffix: %q contains %q", result, suffix)
					}
				}

				// Check that we're not exposing too much of the middle
				if len(tt.token) > 20 {
					middle := tt.token[10:20]
					if strings.Contains(result, middle) {
						t.Errorf("MaskToken exposed token middle: %q contains %q", result, middle)
					}
				}
			}
		})
	}
}

func TestMaskTokenSecurity(t *testing.T) {
	// Test that the function handles Unicode correctly
	t.Run("unicode token", func(t *testing.T) {
		token := "test_こんにちは世界1234567890"
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
