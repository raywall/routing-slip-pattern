package slip

import "strings"

var defaultSensitiveFields = map[string]struct{}{
	"authorization":   {},
	"client_secret":   {},
	"access_token":    {},
	"refresh_token":   {},
	"password":        {},
	"token":           {},
	"api_key":         {},
	"x-api-key":       {},
	"x-serial-number": {},
}

// RedactSensitive returns a copy of value with common secret fields masked.
func RedactSensitive(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveField(key) {
				out[key] = "***"
				continue
			}
			out[key] = RedactSensitive(item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			if isSensitiveField(key) {
				out[key] = "***"
				continue
			}
			out[key] = item
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = RedactSensitive(item)
		}
		return out
	default:
		return value
	}
}

func isSensitiveField(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if _, ok := defaultSensitiveFields[key]; ok {
		return true
	}
	return strings.Contains(key, "secret") ||
		strings.Contains(key, "password") ||
		strings.Contains(key, "token")
}
