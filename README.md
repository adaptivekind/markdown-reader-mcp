# Markdown Reader MCP Server

A Model Context Protocol (MCP) server finds and reads Markdown files in
configured directories. This server guarantees **READ-ONLY** access to **ONLY
MARKDOWN** documents, i.e. files ending with **.md** extension.

## TL;DR

**Quick Start:**

1. Build: `go build`
2. Configure: Create `~/.config/markdown-reader-mcp/markdown-reader-mcp.json`
   with your directories you'd like to be discovered through this server.
3. Add to you AI tooling, for example with Claude Code: `claude mcp add
markdown-reader -- markdown-reader-mcp`
4. Use: Ask to "find markdown files" or "read README"

## Use Case: Master Prompt

A use case of this is the provisioning of a controlled collection of
prompts and context that can be applied for all agents running locally. For
example, this can be achieved by providing a master prompt in a file called
`universal-guidance.md` and then referencing this by prompting.

```txt
Please follow universal.guidance.md from Markdown Reader MCP
```

The agent will discover your master prompt and use it going forward. This
`univeral-guidance.md` prompt can reference other prompts to be loaded.

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
  "directories": ["~/my/notes", "~/projects/docs", "/absolute/path"],
  "max_page_size": 100,
  "debug_logging": false,
  "ignore_dirs": ["\\.git$", "node_modules$", "vendor$"],
  "sse_port": 8080,
  "log_file": "~/local/logs/markdown-reader-mcp.log"
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

The server can be started with SSE transport

```sh
./markdown-reader-mcp -sse
```

and registered with

```sh
claude mcp add --transport sse markdown-reader http://localhost:8080/sse
```

## Run as service in Mac OS with Launchd

The server can be loaded with Launchd on Mac OS

```sh
go install
cp com.adaptivekind.markdown-reader-mcp.plist ~/Library/LaunchAgents
defaults write ~/Library/LaunchAgents/com.adaptivekind.markdown-reader-mcp.plist \
   ProgramArguments -array $HOME/go/bin/markdown-reader-mcp -sse
launchctl load ~/Library/LaunchAgents/com.adaptivekind.markdown-reader-mcp.plist
```

#### Claude Desktop App

Add to your Claude Desktop MCP settings file:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux:** `~/.config/claude-desktop/claude_desktop_config.json`

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
- **`ignore_dirs`** (optional): Regex patterns for directories to ignore.
  Default: `["\\.git$", "node_modules$"]`
- **`sse_port`** (optional): Port for SSE server. Default: 8080
- **`log_file`** (optional): Path to log file. Default: stderr. Supports tilde expansion.

### Directory Filtering

The server automatically ignores common directories that shouldn't contain user documentation:

**Default ignored:**

- `.git` - Version control directories
- `node_modules` - Node.js dependencies

**Custom regex patterns example:**

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

Look for the server in the MCP section of Claude Desktop settings, or check the
logs for startup messages.

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
