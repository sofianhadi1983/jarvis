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
