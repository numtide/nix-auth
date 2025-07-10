package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// ReadSecureInput reads sensitive input (like tokens) from stdin.
// It uses secure password input for terminals and handles non-terminal input gracefully.
func ReadSecureInput(prompt string) (string, error) {
	fmt.Print(prompt)

	// Check if stdin is a terminal
	if term.IsTerminal(int(syscall.Stdin)) {
		// Use secure password input for terminals
		byteInput, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // Add newline after password input
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		return strings.TrimSpace(string(byteInput)), nil
	}

	// For non-terminal input (like tests or piped input)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(strings.TrimSuffix(input, "\n")), nil
}

// ReadInput reads regular input from stdin (non-sensitive).
func ReadInput(prompt string) (string, error) {
	fmt.Print(prompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(strings.TrimSuffix(input, "\n")), nil
}

// ReadYesNo reads a yes/no response from the user.
// Returns true for "y" or "Y", false for anything else.
func ReadYesNo(prompt string) (bool, error) {
	response, err := ReadInput(prompt)
	if err != nil {
		return false, err
	}
	return response == "y" || response == "Y", nil
}
