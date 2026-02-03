# Jarvis

A sophisticated terminal-based AI coding assistant powered by Claude. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a beautiful, responsive TUI experience.

> *"At your service, sir."*

## Demo

<p align="center">
  <img src="demo.gif" alt="Jarvis in Action" width="800">
</p>

## Features

- **Streaming Responses** - Real-time token streaming with elegant rendering
- **Agentic Tools** - Read, edit files, execute commands, fetch web content
- **OAuth & API Key Auth** - Secure authentication via Anthropic OAuth or API key
- **Modular Prompts** - Customizable system prompts with template variables
- **Session History** - Navigate previous commands with arrow keys
- **Beautiful TUI** - Dark theme with syntax highlighting and diff views

## Quick Start

### 1. Build

```bash
go build -o jarvis ./cmd/jarvis
```

### 2. Authenticate

**Option A: OAuth (Recommended)**
```bash
./jarvis
# Select "Login with Claude" and follow browser prompts
```

**Option B: API Key**
```bash
cp .env.example .env
# Add your key: ANTHROPIC_API_KEY=sk-ant-...
```

### 3. Run

```bash
./jarvis
```

## Configuration

Customize behavior in `config.yaml`:

```yaml
anthropic:
  model: "claude-sonnet-4-5-20250929"
  max_tokens: 32000

app:
  name: "Jarvis"
  prompt:
    file: "agent.system.main.md"
    variables:
      agent_name: "Jarvis"
      role: "a sophisticated AI coding assistant"

ui:
  theme: "dark"
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Submit message |
| `↑` / `↓` | Navigate history |
| `Esc` | Interrupt process |
| `Ctrl+C` | Quit |

## Commands

| Command | Description |
|---------|-------------|
| `/clear` | Clear chat history |
| `exit` | Exit application |

## Tools

| Tool | Description |
|------|-------------|
| **Read** | Read file contents |
| **ListFiles** | List directory tree |
| **Update** | Edit files with precise text replacement |
| **Bash** | Execute shell commands |
| **Fetch** | Fetch and extract web content |

## Project Structure

```
jarvis/
├── cmd/jarvis/         # Entry point
├── internal/
│   ├── agent/          # Claude streaming agent
│   ├── auth/           # OAuth & API key auth
│   ├── config/         # YAML configuration
│   ├── prompt/         # Template loader
│   ├── registry/       # Tool registry
│   └── tui/            # Bubble Tea UI
├── pkg/tools/          # Tool implementations
├── prompts/            # System prompt templates
└── config.yaml
```

## Development

```bash
go test ./...      # Run tests
go build ./...     # Build
go run ./cmd/jarvis # Run directly
```

## License

MIT
