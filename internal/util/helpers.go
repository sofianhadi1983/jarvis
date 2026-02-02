package util

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
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

func ClearAPIKeyFromEnv(envPath string) error {
	file, err := os.Open(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "ANTHROPIC_API_KEY=") {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
