package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"jarvis/internal/agent"
	"jarvis/internal/auth"
	"jarvis/internal/config"
	mcpclient "jarvis/internal/mcp"
	"jarvis/internal/registry"
	"jarvis/internal/skills"
	"jarvis/internal/tui"
	"jarvis/pkg/tools"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/sofianhadi1983/anthropic-sdk-go"
	"github.com/sofianhadi1983/anthropic-sdk-go/oauth"
	"github.com/sofianhadi1983/anthropic-sdk-go/option"
)

// claudeCodeTransport ensures correct User-Agent for Claude Code credentials
type claudeCodeTransport struct {
	transport http.RoundTripper
}

func (t *claudeCodeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Override User-Agent to Claude Code's required value
	req.Header.Set("User-Agent", "claude-cli/2.1.2 (external, cli)")
	if t.transport == nil {
		t.transport = http.DefaultTransport
	}
	return t.transport.RoundTrip(req)
}

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	reg := registry.NewRegistry()
	registerTools(reg)

	// Initialize MCP servers if configured
	var mcpManager *mcpclient.Manager
	if len(cfg.MCP.Servers) > 0 {
		mcpManager = mcpclient.NewManager(cfg.MCP.Servers)
		ctx := context.Background()
		if err := mcpManager.Connect(ctx); err != nil {
			log.Printf("Warning: MCP connection error: %v", err)
		}
		defer mcpManager.Close()

		for _, tool := range mcpManager.GetTools(ctx) {
			reg.RegisterOrReplace(tool)
		}
	}

	agentFactory := func(apiKey string) (tui.AgentInterface, error) {
		var client anthropic.Client
		if apiKey != "" {
			client = anthropic.NewClient(option.WithAPIKey(apiKey))
		} else {
			client = anthropic.NewClient(oauth.WithLoadEnv())
		}
		// client := anthropic.NewClient(oauth.WithLoadEnv())
		return agent.NewAgent(client, reg, cfg)
	}

	skillLoader := skills.NewSkillLoader("skills")

	var ag tui.AgentInterface
	needsLogin := true
	envPath := ".env"

	authType := auth.GetAuthType(envPath)

	switch authType {
	case "apikey":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		var err error
		ag, err = agentFactory(apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
			os.Exit(1)
		}
		needsLogin = false

	case "oauth":
		tokens, _ := auth.GetOAuthTokens(envPath)
		if tokens != nil && !auth.IsTokenExpired(tokens.ExpiresAt) {
			// Token is still valid
			os.Setenv("ANTHROPIC_ACCESS_TOKEN", tokens.AccessToken)
			var err error
			ag, err = agentFactory("")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create agent: %v\n", err)
				os.Exit(1)
			}
			needsLogin = false
		} else if tokens != nil && tokens.RefreshToken != "" {
			// Token expired, try to refresh
			newTokens, err := auth.RefreshAccessToken(tokens.RefreshToken)
			if err == nil {
				auth.SaveOAuthTokens(envPath, newTokens)
				os.Setenv("ANTHROPIC_ACCESS_TOKEN", newTokens.AccessToken)
				ag, _ = agentFactory("")
				needsLogin = false
			}
			// If refresh fails, user will need to re-authenticate
		}
	}

	var mcpProvider tui.MCPStatusProvider
	if mcpManager != nil {
		mcpProvider = mcpManager
	}
	model := tui.New(ag, agentFactory, cfg, needsLogin, skillLoader, mcpProvider)

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
