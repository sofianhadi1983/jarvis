package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveAndGetOAuthTokens(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envPath := filepath.Join(tmpDir, ".env")

	tokens := &TokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}

	// Save tokens
	err = SaveOAuthTokens(envPath, tokens)
	if err != nil {
		t.Fatalf("SaveOAuthTokens failed: %v", err)
	}

	// Retrieve tokens
	retrieved, err := GetOAuthTokens(envPath)
	if err != nil {
		t.Fatalf("GetOAuthTokens failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected tokens, got nil")
	}

	if retrieved.AccessToken != "test_access_token" {
		t.Errorf("Expected access token 'test_access_token', got '%s'", retrieved.AccessToken)
	}
	if retrieved.RefreshToken != "test_refresh_token" {
		t.Errorf("Expected refresh token 'test_refresh_token', got '%s'", retrieved.RefreshToken)
	}

	// ExpiresAt should be roughly now + 3600 seconds
	expectedExpires := time.Now().Unix() + 3600
	if retrieved.ExpiresAt < expectedExpires-5 || retrieved.ExpiresAt > expectedExpires+5 {
		t.Errorf("ExpiresAt should be around %d, got %d", expectedExpires, retrieved.ExpiresAt)
	}
}

func TestGetOAuthTokensNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envPath := filepath.Join(tmpDir, ".env_nonexistent")

	tokens, err := GetOAuthTokens(envPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tokens != nil {
		t.Error("Expected nil tokens for non-existent file")
	}
}

func TestGetOAuthTokensEmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte(""), 0600)

	tokens, err := GetOAuthTokens(envPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if tokens != nil {
		t.Error("Expected nil tokens for empty file")
	}
}

func TestClearAllTokens(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envPath := filepath.Join(tmpDir, ".env")

	// First save some tokens
	tokens := &TokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresIn:    3600,
	}
	SaveOAuthTokens(envPath, tokens)

	// Also add API key manually to test it gets cleared
	content, _ := os.ReadFile(envPath)
	content = append(content, []byte("\nANTHROPIC_API_KEY=sk-test-key\n")...)
	os.WriteFile(envPath, content, 0600)

	// Clear all tokens
	err = ClearAllTokens(envPath)
	if err != nil {
		t.Fatalf("ClearAllTokens failed: %v", err)
	}

	// Verify tokens are cleared
	retrieved, _ := GetOAuthTokens(envPath)
	if retrieved != nil {
		t.Error("Expected nil tokens after clearing")
	}

	// Verify API key is also cleared
	authType := GetAuthType(envPath)
	if authType != "" {
		t.Errorf("Expected empty auth type after clearing, got '%s'", authType)
	}
}

func TestGetAuthType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name         string
		envContent   string
		expectedType string
	}{
		{
			name:         "no credentials",
			envContent:   "",
			expectedType: "",
		},
		{
			name:         "api key only",
			envContent:   "ANTHROPIC_API_KEY=sk-test-key",
			expectedType: "apikey",
		},
		{
			name:         "oauth only",
			envContent:   "ANTHROPIC_ACCESS_TOKEN=test_token",
			expectedType: "oauth",
		},
		{
			name:         "both - oauth takes precedence",
			envContent:   "ANTHROPIC_API_KEY=sk-test-key\nANTHROPIC_ACCESS_TOKEN=test_token",
			expectedType: "oauth",
		},
		{
			name:         "empty api key",
			envContent:   "ANTHROPIC_API_KEY=",
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envPath := filepath.Join(tmpDir, tt.name+".env")
			os.WriteFile(envPath, []byte(tt.envContent), 0600)

			authType := GetAuthType(envPath)
			if authType != tt.expectedType {
				t.Errorf("Expected auth type '%s', got '%s'", tt.expectedType, authType)
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		expected  bool
	}{
		{
			name:      "expired in past",
			expiresAt: time.Now().Unix() - 100,
			expected:  true,
		},
		{
			name:      "expires within 5 minutes",
			expiresAt: time.Now().Unix() + 60, // 1 minute from now
			expected:  true,
		},
		{
			name:      "expires in more than 5 minutes",
			expiresAt: time.Now().Unix() + 600, // 10 minutes from now
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTokenExpired(tt.expiresAt)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSetEnvValueUpdatesExisting(t *testing.T) {
	lines := []string{"KEY1=value1", "KEY2=value2"}
	result := setEnvValue(lines, "KEY1", "newvalue1")

	if len(result) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(result))
	}
	if result[0] != "KEY1=newvalue1" {
		t.Errorf("Expected 'KEY1=newvalue1', got '%s'", result[0])
	}
}

func TestSetEnvValueAddsNew(t *testing.T) {
	lines := []string{"KEY1=value1"}
	result := setEnvValue(lines, "KEY2", "value2")

	if len(result) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(result))
	}
	found := false
	for _, line := range result {
		if line == "KEY2=value2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'KEY2=value2' in result")
	}
}

func TestRemoveEnvKey(t *testing.T) {
	lines := []string{"KEY1=value1", "KEY2=value2", "KEY3=value3"}
	result := removeEnvKey(lines, "KEY2")

	if len(result) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(result))
	}
	for _, line := range result {
		if strings.HasPrefix(line, "KEY2=") {
			t.Error("KEY2 should have been removed")
		}
	}
}

func TestSaveOAuthTokensRemovesAPIKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envPath := filepath.Join(tmpDir, ".env")

	// Start with an API key
	os.WriteFile(envPath, []byte("ANTHROPIC_API_KEY=sk-test-key\n"), 0600)

	tokens := &TokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresIn:    3600,
	}

	err = SaveOAuthTokens(envPath, tokens)
	if err != nil {
		t.Fatalf("SaveOAuthTokens failed: %v", err)
	}

	// Verify API key was removed
	content, _ := os.ReadFile(envPath)
	if strings.Contains(string(content), "ANTHROPIC_API_KEY") {
		t.Error("API key should have been removed when saving OAuth tokens")
	}
}
