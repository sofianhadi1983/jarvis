package config

import (
	"fmt"
	"os"

	"jarvis/internal/prompt"

	"gopkg.in/yaml.v3"
)

type PromptConfig struct {
	Files     []string          `yaml:"files,omitempty"`
	Variables map[string]string `yaml:"variables,omitempty"`
	File      string            `yaml:"file,omitempty"`
}

type Config struct {
	Anthropic struct {
		Model     string `yaml:"model"`
		MaxTokens int64  `yaml:"max_tokens"`
	} `yaml:"anthropic"`

	App struct {
		Name         string       `yaml:"name"`
		Prompt       PromptConfig `yaml:"prompt"`
		SystemPrompt string       `yaml:"system_prompt,omitempty"`
	} `yaml:"app"`

	UI struct {
		Theme string `yaml:"theme"`
	} `yaml:"ui"`
}

func LoadConfig() (*Config, error) {
	raw, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(raw, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	if cfg.Anthropic.Model == "" {
		cfg.Anthropic.Model = "claude-sonnet-4-5-20250929"
	}
	if cfg.Anthropic.MaxTokens == 0 {
		cfg.Anthropic.MaxTokens = 8192
	}
	if cfg.App.Name == "" {
		cfg.App.Name = "Jarvis"
	}
	if cfg.UI.Theme == "" {
		cfg.UI.Theme = "dark"
	}

	return &cfg, nil
}

func (c *Config) LoadSystemPrompt() (string, error) {
	loader := prompt.NewLoader("./prompts")

	if c.App.SystemPrompt != "" {
		return c.App.SystemPrompt, nil
	}

	var rawPrompt string
	var err error

	if len(c.App.Prompt.Files) > 0 {
		rawPrompt, err = loader.LoadPrompt(c.App.Prompt.Files)
	} else if c.App.Prompt.File != "" {
		rawPrompt, err = loader.LoadSinglePrompt(c.App.Prompt.File)
	} else {
		return "", fmt.Errorf("no prompt configuration found: specify 'prompt.files', 'prompt.file', or 'system_prompt'")
	}

	if err != nil {
		return "", err
	}

	variables := prompt.GetAutoVariables()
	for k, v := range c.App.Prompt.Variables {
		if v == "{{AUTO}}" {
			continue
		}
		variables[k] = v
	}

	finalPrompt := prompt.ApplyVariables(rawPrompt, variables)
	return finalPrompt, nil
}
