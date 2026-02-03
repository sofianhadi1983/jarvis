package auth

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Token storage keys in .env file
const (
	KeyAccessToken  = "ANTHROPIC_ACCESS_TOKEN"
	KeyRefreshToken = "ANTHROPIC_REFRESH_TOKEN"
	KeyExpiresAt    = "ANTHROPIC_TOKEN_EXPIRES_AT"
	KeyAPIKey       = "ANTHROPIC_API_KEY"
)

// TokenData holds the stored OAuth tokens.
type TokenData struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// SaveOAuthTokens saves OAuth tokens to the .env file.
func SaveOAuthTokens(envPath string, tokens *TokenResponse) error {
	// Calculate expiration timestamp
	expiresAt := time.Now().Unix() + tokens.ExpiresIn

	lines := readEnvLines(envPath)
	lines = setEnvValue(lines, KeyAccessToken, tokens.AccessToken)
	lines = setEnvValue(lines, KeyRefreshToken, tokens.RefreshToken)
	lines = setEnvValue(lines, KeyExpiresAt, strconv.FormatInt(expiresAt, 10))

	// Remove API key when using OAuth
	lines = removeEnvKey(lines, KeyAPIKey)

	return writeEnvLines(envPath, lines)
}

// GetOAuthTokens retrieves OAuth tokens from the .env file.
func GetOAuthTokens(envPath string) (*TokenData, error) {
	file, err := os.Open(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	data := &TokenData{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, KeyAccessToken+"=") {
			data.AccessToken = strings.TrimPrefix(line, KeyAccessToken+"=")
		} else if strings.HasPrefix(line, KeyRefreshToken+"=") {
			data.RefreshToken = strings.TrimPrefix(line, KeyRefreshToken+"=")
		} else if strings.HasPrefix(line, KeyExpiresAt+"=") {
			val := strings.TrimPrefix(line, KeyExpiresAt+"=")
			data.ExpiresAt, _ = strconv.ParseInt(val, 10, 64)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return nil if no tokens found
	if data.AccessToken == "" {
		return nil, nil
	}

	return data, nil
}

// ClearAllTokens removes API key and all OAuth tokens from the .env file.
func ClearAllTokens(envPath string) error {
	lines := readEnvLines(envPath)
	lines = removeEnvKey(lines, KeyAPIKey)
	lines = removeEnvKey(lines, KeyAccessToken)
	lines = removeEnvKey(lines, KeyRefreshToken)
	lines = removeEnvKey(lines, KeyExpiresAt)

	return writeEnvLines(envPath, lines)
}

// GetAuthType returns the authentication type based on stored credentials.
// Returns "apikey", "oauth", or "" if no credentials found.
func GetAuthType(envPath string) string {
	file, err := os.Open(envPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hasAPIKey := false
	hasOAuthToken := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, KeyAPIKey+"=") {
			val := strings.TrimPrefix(line, KeyAPIKey+"=")
			if val != "" {
				hasAPIKey = true
			}
		} else if strings.HasPrefix(line, KeyAccessToken+"=") {
			val := strings.TrimPrefix(line, KeyAccessToken+"=")
			if val != "" {
				hasOAuthToken = true
			}
		}
	}

	// OAuth takes precedence if both are present (shouldn't happen normally)
	if hasOAuthToken {
		return "oauth"
	}
	if hasAPIKey {
		return "apikey"
	}
	return ""
}

// IsTokenExpired checks if the token is expired or will expire within 5 minutes.
func IsTokenExpired(expiresAt int64) bool {
	// Check with 5-minute buffer
	bufferSeconds := int64(5 * 60)
	return time.Now().Unix()+bufferSeconds >= expiresAt
}

// readEnvLines reads all lines from the .env file.
func readEnvLines(envPath string) []string {
	file, err := os.Open(envPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// setEnvValue sets or updates a key-value pair in the lines slice.
func setEnvValue(lines []string, key, value string) []string {
	prefix := key + "="
	found := false

	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + value
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, prefix+value)
	}

	return lines
}

// removeEnvKey removes a key from the lines slice.
func removeEnvKey(lines []string, key string) []string {
	prefix := key + "="
	var result []string

	for _, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			result = append(result, line)
		}
	}

	return result
}

// writeEnvLines writes lines to the .env file.
func writeEnvLines(envPath string, lines []string) error {
	// Filter out empty lines at the end
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}

	return os.WriteFile(envPath, []byte(content), 0600)
}
