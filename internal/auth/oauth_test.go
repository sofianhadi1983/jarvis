package auth

import (
	"strings"
	"testing"
)

func TestGenerateOAuthState(t *testing.T) {
	state, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("GenerateOAuthState failed: %v", err)
	}

	if state.CodeVerifier != CodeVerifier {
		t.Errorf("CodeVerifier should be '%s', got '%s'", CodeVerifier, state.CodeVerifier)
	}
	if state.State != CodeVerifier {
		t.Errorf("State should equal CodeVerifier '%s', got '%s'", CodeVerifier, state.State)
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	state, _ := GenerateOAuthState()
	url := BuildAuthorizationURL(state)

	expectedURL := "https://claude.ai/oauth/authorize?code=true&client_id=9d1c250a-e61b-44d9-88ed-5944d1962f5e&response_type=code&redirect_uri=https%3A%2F%2Fconsole.anthropic.com%2Foauth%2Fcode%2Fcallback&scope=org%3Acreate_api_key+user%3Aprofile+user%3Ainference&code_challenge=OD6HWMArsI00iHxSQ1ioc7Dwxt0OSUdxVui0MpUBRiQ&code_challenge_method=S256&state=Iw-Y0UgiIxn7p2wr3qaDIRaDYBGo21I9EvDW-Cr2lg2lfs2zftOyQL1PWNOpDb8-M3Jm2xflSfvqO5WIhDB5qw"

	if url != expectedURL {
		t.Errorf("URL mismatch.\nExpected: %s\nGot: %s", expectedURL, url)
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
