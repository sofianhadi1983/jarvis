package util

import (
	"encoding/json"
)

func GetTimeoutOrDefault(timeout, defaultValue int) int {
	if timeout > 0 {
		return timeout
	}
	return defaultValue
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func FormatJSON(data any) string {
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(formatted)
}
