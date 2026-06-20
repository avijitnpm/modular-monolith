package logger

import (
	"regexp"
	"strings"
)

var SensitiveKeys = []string{
	"authorization",
	"cookie",
	"set-cookie",
	"token",
	"access_token",
	"refresh_token",
	"secret",
	"client_secret",
	"api_key",
	"password",
	"webhook_secret",
}

var (
	jwtRegex   = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

func MaskEmail(value string) string {
	if value == "" {
		return ""
	}
	at := strings.Index(value, "@")
	if at < 1 {
		return value
	}
	return value[:1] + "***" + value[at:]
}

func MaskToken(value string) string {
	if value == "" {
		return ""
	}
	return "[REDACTED]"
}

func MaskJWT(value string) string {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "eyJ") && strings.Count(value, ".") == 2 {
		return "[REDACTED_JWT]"
	}
	return value
}

func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	return "[REDACTED]"
}

func MaskAuthorizationHeader(value string) string {
	if value == "" {
		return ""
	}
	if i := strings.IndexByte(value, ' '); i > 0 {
		return value[:i] + " [REDACTED]"
	}
	return "[REDACTED]"
}

func RedactString(value string) string {
	result := jwtRegex.ReplaceAllString(value, "[REDACTED_JWT]")
	if strings.Contains(result, "Bearer ") {
		result = regexp.MustCompile(`Bearer\s+\S+`).ReplaceAllString(result, "Bearer [REDACTED]")
	}
	result = emailRegex.ReplaceAllStringFunc(result, MaskEmail)
	return result
}

func RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if isSensitiveKey(k) {
			out[k] = "[REDACTED]"
		} else {
			out[k] = v
		}
	}
	return out
}

func isSensitiveKey(key string) bool {
	for _, s := range SensitiveKeys {
		if strings.EqualFold(key, s) {
			return true
		}
	}
	return false
}
