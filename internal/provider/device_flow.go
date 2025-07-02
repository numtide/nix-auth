package provider

import (
	"fmt"

	"github.com/cli/browser"
)

// DisplayDeviceCode shows the device code and prompts the user to copy it
func DisplayDeviceCode(code string) {
	fmt.Println()
	fmt.Printf("One-time code: %s\n", code)
	fmt.Println()
	fmt.Printf("Copy the code above and press Enter to continue...")
	fmt.Scanln()
}

// DisplayURLAndOpenBrowser shows the authorization URL and attempts to open it in the browser
func DisplayURLAndOpenBrowser(url string) {
	fmt.Println()
	fmt.Printf("Authorization URL: %s\n", url)
	fmt.Println()
	fmt.Println("Opening browser...")
	if err := browser.OpenURL(url); err != nil {
		fmt.Println("Could not open browser automatically.")
		fmt.Println("Please manually visit the URL above and enter your code.")
	}
}

// ShowWaitingMessage displays a waiting message for authorization
func ShowWaitingMessage() {
	fmt.Println()
	fmt.Println("Waiting for authorization...")
}
