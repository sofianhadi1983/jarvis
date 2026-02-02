package main

import (
	"fmt"
	"os"

	"jarvis/internal/agent"
	"jarvis/internal/config"
	"jarvis/internal/registry"
	"jarvis/internal/tui"
	"jarvis/pkg/tools"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"

	"github.com/sofianhadi1983/anthropic-sdk-go"
	"github.com/sofianhadi1983/anthropic-sdk-go/option"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY environment variable is required")
		os.Exit(1)
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	reg := registry.NewRegistry()
	registerTools(reg)

	ag, err := agent.NewAgent(&client, reg, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	model := tui.New(ag, cfg)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	tui.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func registerTools(reg *registry.Registry) {
	reg.Register(tools.ReadFileDefinition)
	reg.Register(tools.ListFilesDefinition)
	reg.Register(tools.UpdateFileDefinition)
	reg.Register(tools.BashDefinition)
	reg.Register(tools.FetchDefinition)
}
