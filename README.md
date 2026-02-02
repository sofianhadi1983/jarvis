# Jarvis

A sophisticated terminal-based AI coding assistant with a beautiful TUI built on Bubble Tea. Think of it as your personal AI butler for coding - refined, articulate, and always at your service.

## Features

- Interactive terminal UI with streaming responses
- Jarvis-style communication - formal, witty, and always helpful
- Explains complex concepts with real-world analogies
- Tool execution: Read, ListFiles, Update, Bash, Fetch
- Modular prompt system with variable substitution
- Command history with session management
- YAML-based configuration

## Installation

```bash
go build -o jarvis ./cmd/chewbacca
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
  model: "claude-sonnet-4-5-20250929"
  max_tokens: 8192

app:
  name: "Jarvis"
  prompt:
    file: "agent.system.main.md"
    variables:
      agent_name: "Jarvis"
      role: "a sophisticated AI coding assistant, much like the trusted companion to Mr. Stark"

ui:
  theme: "dark"
```

## Usage

```bash
./jarvis
```

### Keyboard Shortcuts

- `Enter` - Submit message
- `Up/Down` - Navigate command history
- `Esc` - Interrupt running process
- `Ctrl+C` - Quit

### Commands

- `/clear` or `clear` - Clear chat history and context
- `exit` or `quit` - Exit the application

## Project Structure

```
jarvis/
├── cmd/chewbacca/      # CLI entry point
├── internal/
│   ├── agent/          # Core agent with streaming
│   ├── config/         # Configuration management
│   ├── history/        # Command history management
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

## Personality

Jarvis communicates with refined eloquence and dry wit:
- Addresses users respectfully ("Sir", "If I may suggest...")
- Offers real-world analogies for complex technical concepts
- Always verifies understanding after explaining difficult topics
- Maintains professionalism with a subtle sense of humor
