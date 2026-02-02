package prompt

import (
	"strings"
	"time"
)

type Variables map[string]string

func GetAutoVariables() Variables {
	return Variables{
		"current_date":     time.Now().Format("2006-01-02"),
		"current_time":     time.Now().Format("15:04:05"),
		"current_datetime": time.Now().Format("2006-01-02 15:04:05"),
	}
}

func ApplyVariables(prompt string, vars Variables) string {
	result := prompt

	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}
