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
- **Subagents** - Spawn isolated agents for parallel exploration, coding, and planning tasks
- **MCP Integration** - Connect to any MCP-compatible server and use its tools seamlessly
- **Skills** - Extensible skill system with slash commands (e.g. `/commit`)
- **Todo Tracking** - Built-in task management for multi-step workflows
- **OAuth & API Key Auth** - Secure authentication via Anthropic OAuth or API key
- **Modular Prompts** - Customizable system prompts with template variables
- **Session History** - Navigate previous commands with arrow keys
- **Image Support** - Send images alongside text messages
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

### MCP Servers

Connect to external [Model Context Protocol](https://modelcontextprotocol.io/) servers by adding an `mcp` section to `config.yaml`. MCP tools are discovered at startup and become available to the agent alongside built-in tools.

```yaml
mcp:
  servers:
    filesystem:
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    github:
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-github"]
      env:
        GITHUB_TOKEN: "your-token"
```

Each server is launched as a subprocess using stdio transport. Tools are namespaced as `mcp_{server}_{tool}` to avoid collisions with built-in tools. If a server fails to start, Jarvis logs a warning and continues with the remaining servers.

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
| `/commit` | Create a git commit with conventional format |
| `exit` | Exit application |

## Tools

### Built-in

| Tool | Description |
|------|-------------|
| **Read** | Read file contents |
| **ListFiles** | List directory tree |
| **Update** | Edit files with precise text replacement |
| **Bash** | Execute shell commands |
| **Fetch** | Fetch and extract web content |
| **TodoWrite** | Track multi-step tasks with status |
| **Task** | Spawn subagents (explore, code, plan) |
| **Skill** | Load specialized skills |

### MCP Tools

Any tools exposed by configured MCP servers are automatically registered and available to the agent. They appear with the naming convention `mcp_{server}_{tool}`.

## Skills

Skills are extensible plugins defined as Markdown files in the `skills/` directory. Each skill directory contains a `SKILL.md` with YAML frontmatter:

```yaml
---
name: commit
description: Create well-formatted git commits
user_invocable: true
---
# Instructions for the agent...
```

User-invocable skills can be triggered with slash commands (e.g. `/commit`).

## Project Structure

```
jarvis/
├── cmd/jarvis/         # Entry point
├── internal/
│   ├── agent/          # Claude streaming agent
│   ├── auth/           # OAuth & API key auth
│   ├── config/         # YAML configuration
│   ├── mcp/            # MCP client integration
│   ├── prompt/         # Template loader
│   ├── registry/       # Tool registry
│   ├── skills/         # Skill loader
│   ├── subagent/       # Subagent task execution
│   ├── todo/           # Todo tracking
│   └── tui/            # Bubble Tea UI
├── pkg/tools/          # Tool implementations
├── prompts/            # System prompt templates
├── skills/             # Skill definitions
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
