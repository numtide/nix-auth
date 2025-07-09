package util

import "fmt"

// MaskToken masks a token for security, showing only the first and last 4 characters
func MaskToken(token string) string {
	if len(token) > 10 {
		return fmt.Sprintf("%s****%s", token[:4], token[len(token)-4:])
	}
	return "Configured"
}
