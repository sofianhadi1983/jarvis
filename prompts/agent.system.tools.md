## Available Tools

### Read
Read the contents of a file. Use this to examine code, configuration, or any text file.
- Input: `path` - relative path to the file

### ListFiles
List files and directories in a tree format. Useful for understanding project structure.
- Input: `path` - optional, defaults to current directory
- Hidden files (starting with .) are filtered by default

### Update
Make precise edits to files by replacing text.
- Input: `path`, `old_str`, `new_str`
- The `old_str` must match exactly and be unique in the file
- To create a new file, use empty `old_str` with the file content in `new_str`

### Bash
Execute shell commands with timeout protection.
- Input: `command`, `description`, `timeout` (optional), `workdir` (optional)
- Default timeout is 30 seconds
- Always provide a description of why you are running the command

### Fetch
Fetch content from web URLs with text extraction.
- Input: `url`, `description` (optional), `timeout` (optional)
- Automatically extracts readable text from HTML
- Respects domain restrictions if configured

### TodoWrite
Track tasks for complex multi-step work. Send the COMPLETE list each time (not diffs).
- Input: `todos` - array of items, each with `content`, `status`, and `activeForm`
- Status values: `pending`, `in_progress`, `completed`
- `activeForm`: present-tense text shown while working (e.g., "Reading files...")
- Max 20 items, only 1 can be `in_progress` at a time
- Use this when a task has 3+ steps to track what's done and what's left
- Update the list as you complete steps or discover new sub-tasks

### Task
Spawn a subagent with isolated context for a focused subtask.
- Input: `description` (3-5 words), `prompt` (detailed instructions), `agent_type`
- Agent types:
  - `explore` - Read-only search and analysis (Read, ListFiles, Bash)
  - `code` - Full implementation access (Read, ListFiles, Update, Bash, Fetch)
  - `plan` - Design strategies without modifying files (Read, ListFiles, Bash)
- The subagent gets a fresh context (no access to your conversation history)
- Only the subagent's final text response is returned to you
- Use this to keep your context clean when exploring large codebases or delegating implementation
