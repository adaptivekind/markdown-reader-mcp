# Markdown Reader MCP Server

A Model Context Protocol (MCP) server built in Go that **reads** markdown
documents from configured local directories. The scope of this MCP server is
explicitly constrained to read-only to minimise security concerns.

## Features

- **Resource Access**: Access markdown files as MCP resources
- **Directory Scanning**: Automatically discovers markdown files in specified directories
- **Search Functionality**: Search for text within markdown files
- **Command Line Configuration**: Specify directories via command line arguments
- **Security**: Path validation to prevent directory traversal attacks

## Resources

### `markdown://list`

Returns a JSON list of all markdown files found in configured directories,
including file metadata.

## Tools

### `read_markdown_file`

Read the content of a specific markdown file by providing its file path.

## Usage

Run the server with directories specified as a comma-separated list:

```bash
# Scan current directory only (default)
./markdown-reader-mcp

# Scan multiple directories
./markdown-reader-mcp -dirs "/path/to/docs,/another/path,./local/docs"

# Scan current directory and test_data
./markdown-reader-mcp -dirs ".,./test_data"
```

The server communicates via stdio using the MCP protocol.

## Integration with Claude Code

To use this MCP server with Claude Code, you need to configure it in your MCP settings.

### Step 1: Build the Server

```bash
go build -o markdown-reader-mcp .
```

### Step 2: Configure Claude Code

Add the server to your Claude Code MCP configuration. The configuration location depends on your setup:

**Option A: Using CLAUDE.md (Recommended)**

Create or update `CLAUDE.md` in your project root:

```markdown
# MCP Servers

## Markdown Reader

- **Command**: `./markdown-reader-mcp`
- **Args**: `["-dirs", "docs,guides,README.md"]`
- **Description**: Reads and searches markdown files in specified directories
```

**Option B: Using MCP Configuration File**

Create or update your MCP configuration file (typically `~/.config/claude-code/mcp.json` or similar):

```json
{
  "mcpServers": {
    "markdown-reader": {
      "command": "/absolute/path/to/markdown-reader-mcp",
      "args": ["-dirs", "docs,guides,./"],
      "env": {}
    }
  }
}
```

### Step 3: Available Capabilities

Once configured, Claude Code will have access to:

**Resources:**

- `markdown://list` - Get a JSON list of all markdown files with metadata

**Tools:**

- `read_markdown_file` - Read content of specific markdown files
- `search_markdown` - Search for text within markdown files
- `list_directories` - List configured directories

### Step 4: Example Usage

After setup, you can ask Claude Code to:

- "Show me all markdown files in the project"
- "Read the content of docs/api.md"
- "Search for 'authentication' across all markdown files"
- "What directories are being scanned for markdown files?"

### Configuration Tips

- Use absolute paths in MCP configuration for reliability
- Specify relevant directories with `-dirs` to limit scope and improve performance
- Include common documentation directories like `docs/`, `guides/`, or project root
- The server automatically discovers `.md` files recursively in specified directories

### Troubleshooting

**Server not starting:**

- Ensure the binary has execute permissions: `chmod +x markdown-reader-mcp`
- Verify the path in your MCP configuration is correct
- Check that specified directories exist and are readable

**No markdown files found:**

- Verify directories exist: `ls -la your/directory/path`
- Check directory permissions
- Test manually: `./markdown-reader-mcp -dirs "your/dirs" -help`

**Connection issues:**

- Ensure Claude Code can access the binary location
- Try using absolute paths in the MCP configuration
- Check Claude Code logs for error messages

## Development Setup

1. Clone the repository
2. Install pre-commit framework:
   ```bash
   pip install pre-commit
   ```
3. Install the pre-commit hooks:
   ```bash
   pre-commit install
   ```

The pre-commit hooks will automatically run before each commit and include:

- `go fmt` - Code formatting
- `go vet` - Static analysis
- `go mod tidy` - Clean up module dependencies
- `go test` - All tests must pass
- General checks for trailing whitespace, large files, etc.

## Building

```bash
go build -o markdown-reader-mcp .
```
