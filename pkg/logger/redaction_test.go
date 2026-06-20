package logger

import "testing"

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"john@example.com", "j***@example.com"},
		{"a@b.co", "a***@b.co"},
		{"", ""},
		{"noatsign", "noatsign"},
	}
	for _, tt := range tests {
		if got := MaskEmail(tt.input); got != tt.want {
			t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskJWT(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"eyJhbGciOi.eyJzdWIi.signature", "[REDACTED_JWT]"},
		{"notajwt", "notajwt"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := MaskJWT(tt.input); got != tt.want {
			t.Errorf("MaskJWT(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"sometoken123", "[REDACTED]"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := MaskToken(tt.input); got != tt.want {
			t.Errorf("MaskToken(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"secret-value", "[REDACTED]"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := MaskSecret(tt.input); got != tt.want {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskAuthorizationHeader(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Bearer eyJhbGciOi.payload.sig", "Bearer [REDACTED]"},
		{"Basic dXNlcjpwYXNz", "Basic [REDACTED]"},
		{"tokenvalue", "[REDACTED]"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := MaskAuthorizationHeader(tt.input); got != tt.want {
			t.Errorf("MaskAuthorizationHeader(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRedactString(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{
			"token is eyJhbGciOi.eyJzdWIi.abcdef",
			"token is [REDACTED_JWT]",
		},
		{
			"Bearer abc123token",
			"Bearer [REDACTED]",
		},
		{
			"user email john@example.com here",
			"user email j***@example.com here",
		},
		{
			"safe text",
			"safe text",
		},
	}
	for _, tt := range tests {
		if got := RedactString(tt.input); got != tt.want {
			t.Errorf("RedactString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRedactMap(t *testing.T) {
	input := map[string]any{
		"authorization":  "Bearer token123",
		"access_token":   "at_secret",
		"refresh_token":  "rt_secret",
		"api_key":        "key123",
		"webhook_secret": "whsec_abc",
		"name":           "safe-value",
		"path":           "/api/v1/users",
	}

	got := RedactMap(input)

	redacted := []string{"authorization", "access_token", "refresh_token", "api_key", "webhook_secret"}
	for _, k := range redacted {
		if got[k] != "[REDACTED]" {
			t.Errorf("RedactMap[%q] = %v, want [REDACTED]", k, got[k])
		}
	}

	safe := []string{"name", "path"}
	for _, k := range safe {
		if got[k] != input[k] {
			t.Errorf("RedactMap[%q] = %v, want %v", k, got[k], input[k])
		}
	}

	if RedactMap(nil) != nil {
		t.Error("RedactMap(nil) should return nil")
	}
}
