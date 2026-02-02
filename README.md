# Chewbacca

A terminal-based AI coding assistant with a beautiful TUI built on Bubble Tea.

## Features

- Interactive terminal UI with streaming responses
- Tool execution: Read, ListFiles, Update, Bash, Fetch
- Modular prompt system with variable substitution
- YAML-based configuration

## Installation

```bash
go build -o chewbacca ./cmd/chewbacca
```

## Configuration

1. Copy the environment template:
```bash
cp .env.example .env
```

2. Add your Anthropic API key to `.env`:
```
ANTHROPIC_API_KEY=your_api_key_here
```

3. Customize `config.yaml` as needed:
```yaml
anthropic:
  model: "claude-3-5-sonnet-20241022"
  max_tokens: 8192

app:
  name: "Chewbacca"
  prompt:
    file: "agent.system.main.md"
    variables:
      agent_name: "Chewbacca"
      role: "an expert AI coding assistant"

ui:
  theme: "dark"
```

## Usage

```bash
./chewbacca
```

### Keyboard Shortcuts

- `Enter` - Submit message
- `Ctrl+C` - Quit

## Project Structure

```
chewbacca/
├── cmd/chewbacca/      # CLI entry point
├── internal/
│   ├── agent/          # Core agent with streaming
│   ├── config/         # Configuration management
│   ├── prompt/         # Prompt loading and templating
│   ├── registry/       # Tool registry
│   ├── tui/            # Bubble Tea TUI
│   │   └── components/ # TUI components
│   └── util/           # Utility functions
├── pkg/tools/          # Tool implementations
├── prompts/            # System prompt files
├── config.yaml         # Configuration
└── .env                # Environment variables
```

## Available Tools

- **Read**: Read file contents
- **ListFiles**: List directory structure
- **Update**: Edit files with precise replacements
- **Bash**: Execute shell commands
- **Fetch**: Fetch web content

## Development

```bash
# Run tests
go test ./...

# Build
go build ./...

# Run
go run ./cmd/chewbacca
```
