package auth

import (
	"crypto/sha256"
	"net/url"
	"strings"
	"testing"
)

func TestGenerateOAuthState(t *testing.T) {
	state, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("GenerateOAuthState failed: %v", err)
	}

	if len(state.CodeVerifier) < 43 || len(state.CodeVerifier) > 128 {
		t.Errorf("CodeVerifier length %d outside RFC 7636 range [43, 128]", len(state.CodeVerifier))
	}

	if state.CodeChallenge == "" {
		t.Error("CodeChallenge should not be empty")
	}

	if state.State == state.CodeVerifier {
		t.Error("State should not equal CodeVerifier")
	}

	hash := sha256.Sum256([]byte(state.CodeVerifier))
	expectedChallenge := base64URLEncode(hash[:])
	if state.CodeChallenge != expectedChallenge {
		t.Errorf("CodeChallenge mismatch.\nExpected: %s\nGot: %s", expectedChallenge, state.CodeChallenge)
	}
}

func TestGenerateOAuthStateUniqueness(t *testing.T) {
	state1, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("first GenerateOAuthState failed: %v", err)
	}

	state2, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("second GenerateOAuthState failed: %v", err)
	}

	if state1.CodeVerifier == state2.CodeVerifier {
		t.Error("two calls produced the same CodeVerifier")
	}
	if state1.State == state2.State {
		t.Error("two calls produced the same State")
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	state, _ := GenerateOAuthState()
	rawURL := BuildAuthorizationURL(state)

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	if parsed.Scheme != "https" {
		t.Errorf("expected scheme https, got %s", parsed.Scheme)
	}
	if parsed.Host != "claude.ai" {
		t.Errorf("expected host claude.ai, got %s", parsed.Host)
	}
	if parsed.Path != "/oauth/authorize" {
		t.Errorf("expected path /oauth/authorize, got %s", parsed.Path)
	}

	params := parsed.Query()

	expectedParams := map[string]string{
		"code":                  "true",
		"client_id":            ClientID,
		"response_type":        "code",
		"redirect_uri":         RedirectURI,
		"scope":                OAuthScopes,
		"code_challenge":       state.CodeChallenge,
		"code_challenge_method": "S256",
		"state":                state.State,
	}

	for key, expected := range expectedParams {
		got := params.Get(key)
		if got != expected {
			t.Errorf("param %s: expected %q, got %q", key, expected, got)
		}
	}
}

func TestExchangeCodeForTokensStateValidation(t *testing.T) {
	state, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("GenerateOAuthState failed: %v", err)
	}

	_, err = ExchangeCodeForTokens("somecode", "wrong-state", state)
	if err == nil {
		t.Fatal("expected error for mismatched state")
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("expected state mismatch error, got: %v", err)
	}
}

func TestParseAuthCodeValid(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCode  string
		expectedState string
	}{
		{
			name:          "standard format",
			input:         "abc123#xyz789",
			expectedCode:  "abc123",
			expectedState: "xyz789",
		},
		{
			name:          "with whitespace",
			input:         "  abc123#xyz789  ",
			expectedCode:  "abc123",
			expectedState: "xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, state, err := ParseAuthCode(tt.input)
			if err != nil {
				t.Fatalf("ParseAuthCode failed: %v", err)
			}
			if code != tt.expectedCode {
				t.Errorf("Expected code '%s', got '%s'", tt.expectedCode, code)
			}
			if state != tt.expectedState {
				t.Errorf("Expected state '%s', got '%s'", tt.expectedState, state)
			}
		})
	}
}

func TestParseAuthCodeInvalid(t *testing.T) {
	_, _, err := ParseAuthCode("")
	if err == nil {
		t.Error("Expected error for empty input")
	}

	_, _, err = ParseAuthCode("   ")
	if err == nil {
		t.Error("Expected error for whitespace only")
	}
}

func TestParseAuthCodeInvalidNoSeparator(t *testing.T) {
	_, _, err := ParseAuthCode("abc123xyz789")
	if err == nil {
		t.Error("Expected error for input without # separator")
	}
}

func TestBase64URLEncode(t *testing.T) {
	input := []byte{0xff, 0xfe, 0xfd}
	result := base64URLEncode(input)

	if strings.Contains(result, "+") || strings.Contains(result, "/") || strings.Contains(result, "=") {
		t.Errorf("base64URLEncode should produce URL-safe output without padding, got: %s", result)
	}
}
