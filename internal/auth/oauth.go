// Package auth provides OAuth authentication functionality for Anthropic services.
package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

// OAuth constants for Anthropic API
const (
	ClientID     = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	RedirectURI  = "https://console.anthropic.com/oauth/code/callback"
	TokenURL     = "https://console.anthropic.com/v1/oauth/token"
	CodeVerifier = "Iw-Y0UgiIxn7p2wr3qaDIRaDYBGo21I9EvDW-Cr2lg2lfs2zftOyQL1PWNOpDb8-M3Jm2xflSfvqO5WIhDB5qw"
)

// TokenResponse represents the response from the OAuth token endpoint.
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	Organization struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"organization"`
	Account struct {
		UUID         string `json:"uuid"`
		EmailAddress string `json:"email_address"`
	} `json:"account"`
}

// OAuthState holds the PKCE code verifier and state parameter.
type OAuthState struct {
	CodeVerifier string
	State        string
}

// GenerateOAuthState creates OAuthState with the fixed PKCE values.
func GenerateOAuthState() (*OAuthState, error) {
	return &OAuthState{
		CodeVerifier: CodeVerifier,
		State:        CodeVerifier,
	}, nil
}

// BuildAuthorizationURL returns the exact authorization URL.
func BuildAuthorizationURL(oauthState *OAuthState) string {
	return "https://claude.ai/oauth/authorize?code=true&client_id=9d1c250a-e61b-44d9-88ed-5944d1962f5e&response_type=code&redirect_uri=https%3A%2F%2Fconsole.anthropic.com%2Foauth%2Fcode%2Fcallback&scope=org%3Acreate_api_key+user%3Aprofile+user%3Ainference&code_challenge=OD6HWMArsI00iHxSQ1ioc7Dwxt0OSUdxVui0MpUBRiQ&code_challenge_method=S256&state=Iw-Y0UgiIxn7p2wr3qaDIRaDYBGo21I9EvDW-Cr2lg2lfs2zftOyQL1PWNOpDb8-M3Jm2xflSfvqO5WIhDB5qw"
}

// ExchangeCodeForTokens exchanges the authorization code for access and refresh tokens.
func ExchangeCodeForTokens(code, state string, oauthState *OAuthState) (*TokenResponse, error) {
	reqBody := map[string]string{
		"code":          code,
		"state":         state,
		"grant_type":    "authorization_code",
		"client_id":     ClientID,
		"redirect_uri":  RedirectURI,
		"code_verifier": CodeVerifier,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", TokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "claude-cli/2.1.2 (external, cli)")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshAccessToken uses the refresh token to obtain a new access token.
func RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	reqBody := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     ClientID,
		"refresh_token": refreshToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", TokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "claude-cli/2.1.2 (external, cli)")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	return &tokenResp, nil
}

// ParseAuthCode parses authentication code in "code#state" format.
func ParseAuthCode(input string) (code, state string, err error) {
	input = strings.TrimSpace(input)

	if input == "" {
		return "", "", errors.New("authentication code cannot be empty")
	}

	parts := strings.Split(input, "#")
	if len(parts) != 2 {
		return "", "", errors.New("invalid format, expected: code#state")
	}

	return parts[0], parts[1], nil
}

// OpenBrowser opens the specified URL in the default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// base64URLEncode encodes bytes to base64url format without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
