# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **read-only** Model Context Protocol (MCP) server built in Go that enables LLMs to discover, read, and search markdown files in specified directories. The server is explicitly constrained to read-only operations for security.

The server includes optional debug logging (disabled by default) that can be enabled via configuration to track each tool call with input parameters, execution timing, and results for performance monitoring and troubleshooting.

## Build and Development Commands

```bash
# Build the server binary
go build

# Run all tests (includes end-to-end MCP protocol tests)
go test -v

# Run a specific test
go test -v -run TestServerInitialization
```

## Architecture

### Core Components

**Server Setup (`main.go`)**

- Uses `github.com/mark3labs/mcp-go` library for MCP protocol implementation
- Supports directory paths as positional command-line arguments or configuration file
- Registers one resource and one tool with the MCP server
- Communicates via stdio using JSON-RPC 2.0

**File Organization:**

- `main.go`: Server setup, configuration, and MCP server initialization
- `find.go`: File discovery functionality (`findAllMarkdownFiles`, `handleFindAllMarkdownFiles`)
- `read_handler.go`: File reading functionality (`handleReadMarkdownFile`, `findFirstFileByName`)
- `config_test.go`: Tests for configuration file loading functionality

**Key Functions:**

- `findAllMarkdownFiles()` (`find.go`): Recursively discovers `.md` files in configured directories
- `handleFindAllMarkdownFiles()` (`find.go`): Resource handler returning JSON list of all markdown files with metadata
- `handleReadMarkdownFile()` (`read_handler.go`): Tool handler for reading individual file contents by filename only
- `findFirstFileByName()` (`read_handler.go`): Helper function that searches for files by name across all configured directories and returns the first match found
- `loadConfigFromFile()` (`main.go`): Loads configuration from `~/.config/markdown-reader-mcp/markdown-reader-mcp.json`

**Security Model:**

- Path validation prevents directory traversal attacks (`..` sequences blocked)
- File access restricted to configured directories only
- Only `.md` files are discovered and accessible
- Only filename-based search (no path traversal through tool interface)
- All file paths converted to absolute paths for validation

### MCP Interface

**Tools:**

- `find_markdown_files`: Find markdown files with optional query filtering and pagination. Parameters: `query` (optional string to filter by filename), `page_size` (optional number, default 50, max configurable). Returns JSON array of discovered markdown files with metadata (path, name, relativePath) and includes directory list and file count.
- `read_markdown_file`: Read content of specific file by filename only (e.g., 'README' or 'README.md')

### Testing Architecture

**End-to-End Testing (`e2e_test.go`)**

- `MCPTestClient`: Custom test client that spawns server process and communicates via stdio
- Tests full MCP protocol handshake, resource/tool operations, and error handling
- Uses `test/dir1/` and `test/dir2/` directories with sample markdown files for realistic testing
- Includes timeout handling and proper process cleanup

**Unit Testing:**

- `find_test.go`: Tests for file discovery functionality (`TestFindAllMarkdownFiles`, `TestHandleFindAllMarkdown`)
- `read_handler_test.go`: Tests for file reading functionality (`TestHandleReadMarkdownFile`, `TestFindFirstFileByName`)
- `config_test.go`: Tests for configuration file loading (`TestLoadConfigFromFile`, error cases)
- Isolated testing with temporary directories and mock data
- Comprehensive error handling and edge case coverage

## Key Dependencies

- `github.com/mark3labs/mcp-go v0.34.0`: MCP protocol implementation
- Standard library only (no external runtime dependencies)

## Usage Patterns

The server can be configured in two ways:

### Command-line Arguments
```bash
./markdown-reader-mcp docs guides .     # Scan multiple directories
./markdown-reader-mcp /path/to/docs     # Scan specific directory
```

### Configuration File
Create `~/.config/markdown-reader-mcp/markdown-reader-mcp.json`:
```json
{
  "directories": ["~/Documents/notes", "~/projects/docs", "."],
  "max_page_size": 100,
  "debug_logging": true
}
```

Then run without arguments:
```bash
./markdown-reader-mcp                   # Uses config file
```

Command-line arguments take precedence over the configuration file. Tilde (`~`) in directory paths is automatically expanded to the home directory.

Designed to integrate with Claude Code via MCP configuration in `CLAUDE.md`:

```markdown
## Markdown Reader

- **Command**: `./markdown-reader-mcp`
- **Args**: `["docs", "guides", "."]`
```
