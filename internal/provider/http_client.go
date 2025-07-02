package provider

import (
	"context"
	"fmt"
	"net/http"
)

// makeAuthenticatedRequest creates and executes an authenticated HTTP request
// with common error handling for authentication providers
func makeAuthenticatedRequest(ctx context.Context, method, url, token, authHeader string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	// Set authentication header
	req.Header.Set("Authorization", authHeader)

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Check common error status codes
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		resp.Body.Close()
		return nil, fmt.Errorf("token is invalid or expired")
	case http.StatusOK:
		return resp, nil
	default:
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

