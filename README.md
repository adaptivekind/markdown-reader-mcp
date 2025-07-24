# Markdown Reader MCP Server

A Model Context Protocol (MCP) server built in Go with **READ-ONLY** access to
markdown documents from configured local directories. The scope of this MCP
server is explicitly constrained to read-only access, and restricted to only
being able to access files that have the **.md** extension. This explicit constraint helps
with peace of mind.

## TL;DR

**Quick Start:**

1. Build: `go build`
2. Configure: Create `~/.config/markdown-reader-mcp/markdown-reader-mcp.json`
   with your directories
3. Add to Claude Code: `claude mcp add markdown-reader -- markdown-reader-mcp`
4. Add to Claude Desktop: Add to your MCP settings file
5. Use: Ask Claude to "find markdown files" or "read README"

**What it does:**

- 🔍 **Find markdown files** with optional filtering and pagination
- 📖 **Read markdown files** by filename across multiple directories
- 🚫 **Ignores directories** like `.git`, `node_modules` by default

## Use Case: Master Prompt

A primary use case of this is the provisioning of a controlled personal
collection of prompts and context that can be applied for all local agents. For
example, this can be achieved by providing a master prompt in a file called
`my-tone-of-voice.md` and then referencing this by prompting.

```txt
Please use my-tone-of-voice.md from Markdown Reader MCP at all times
```

The agent will discover your master prompt and use it going forward. This
`my-tone-of-voice.md` prompt can reference other prompts, for example, by
including:

```markdown
- You MUST follow guidance in `my-universal-guidance.md`
```

And storing universal guidance in `my-universal-guidance.md`

```markdown
# My universal guidance

## Requirements

- You MUST not use an em dash "—", you can use a normal dash "-".
- You MUST write in a natural and serious voice.
- You MUST NOT add humorous anecdotes.
- You MUST NOT show excessive enthusiasm.

## Finding reference material

- If you need to find information about myself, please see if the material can
  be found in the markdown files from the Markdown Reader MCP.
- If you need to search the web for information, please also search for material
  in the markdown files from the Markdown Reader MCP.
```

This approach can then discover any notes (in your markdown directories) about
yourself as needed.

## Installation & Setup

### 1. Build the Server

```sh
git clone <this-repo>
cd markdown-reader-mcp
go build
```

Install locally:

```sh
go install
```

### 2. Configure Directories

**Option A: Configuration File (Recommended)**

Create `~/.config/markdown-reader-mcp/markdown-reader-mcp.json`:

```json
{
  "directories": ["~/Documents/notes", "~/projects/docs", "/absolute/path"],
  "max_page_size": 100,
  "debug_logging": false,
  "ignore_dirs": ["\\.git$", "node_modules$", "vendor$"]
}
```

**Option B: Command-line Arguments**

```sh
./markdown-reader-mcp ~/documents/notes ~/projects/docs
```

### 3. Integration with Claude

#### Claude Code CLI

```sh
# Add for current project
claude mcp add markdown-reader -- markdown-reader-mcp

# Add globally for all projects
claude mcp add markdown-reader -s user -- markdown-reader-mcp

# Verify installation
claude mcp list

# Remove MCP
claude mcp remove markdown-reader
```

#### Claude Desktop App

Add to your Claude Desktop MCP settings file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**Linux:** `~/.config/claude-desktop/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "markdown-reader": {
      "command": "/path/to/markdown-reader-mcp",
      "args": []
    }
  }
}
```

Or with command-line arguments:

```json
{
  "mcpServers": {
    "markdown-reader": {
      "command": "/path/to/markdown-reader-mcp",
      "args": ["/path/to/your/docs", "/another/path"]
    }
  }
}
```

**Note:** Use absolute paths in Claude Desktop configuration.

## Usage Examples

Once configured, you can ask Claude:

- **"Show me all markdown files in the project"** → Uses `find_markdown_files`
- **"Find files containing 'api' in the name"** → Uses `find_markdown_files` with query
- **"Show me the first 10 markdown files"** → Uses `find_markdown_files` with pagination
- **"Read the content of README"** → Uses `read_markdown_file`
- **"What's in the api.md file?"** → Uses `read_markdown_file`

## Configuration Reference

### Configuration File Options

- **`directories`**: Array of directory paths to scan for markdown files
- **`max_page_size`** (optional): Maximum results per page. Default: 500
- **`debug_logging`** (optional): Enable detailed debug logging. Default: false
- **`ignore_dirs`** (optional): Regex patterns for directories to ignore. Default: `["\\.git$", "node_modules$"]`

### Directory Filtering

The server automatically ignores common directories that shouldn't contain user documentation:

**Default ignored:**

- `.git` - Version control directories
- `node_modules` - Node.js dependencies

**Custom patterns example:**

```json
{
  "ignore_dirs": [
    "\\.git$", // Git directories
    "node_modules$", // Node.js dependencies
    "target$", // Rust/Java build output
    "dist$" // Build output
  ]
}
```

**Pattern syntax:**

- Use Go `regexp` syntax
- End with `$` to match directory name exactly
- Escape dots: `\\.git$` not `.git$`
- Invalid patterns are logged and ignored

## Tools Reference

### `find_markdown_files`

Find markdown files with optional filtering and pagination.

**Parameters:**

- `query` (optional): Filter files by name containing this string
- `page_size` (optional): Limit results (default: 50, max: configurable)

**Returns:** JSON with file list, metadata, and count.

### `read_markdown_file`

Read content of a specific markdown file by filename.

**Parameters:**

- `filename` (required): File name with or without `.md` extension

**Returns:** File content as text.

**Security:** Only accepts filenames (no paths). Searches configured directories automatically.

## Debug Logging

Enable with `"debug_logging": true` in config file.

## Verification

### Claude Code

```sh
# Check MCP server status
claude mcp list

# Test MCP tools in Claude Code
/mcp
```

### Claude Desktop

Look for the server in the MCP section of Claude Desktop settings, or check the logs for startup messages.

### Manual Testing

```sh
# Start server (will show startup logs)
./markdown-reader-mcp ~/your/docs

# Look for:
# "Scanning directories: [...]"
# "Ignoring directories matching patterns: [...]"
# "Starting Markdown Reader MCP server..."
```

## Development

1. **Clone repository**
2. **Install pre-commit** (optional):
   ```sh
   pip install pre-commit
   pre-commit install
   ```
3. **Run tests**: `go test -v`
4. **Build**: `go build`
