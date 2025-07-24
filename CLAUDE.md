# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **read-only** Model Context Protocol (MCP) server built in Go that enables LLMs to discover, read, and search markdown files in specified directories. The server is explicitly constrained to read-only operations for security.

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
- Requires directory paths as positional command-line arguments (no default)
- Registers one resource and one tool with the MCP server
- Communicates via stdio using JSON-RPC 2.0

**File Organization:**

- `main.go`: Server setup, configuration, and MCP server initialization
- `find.go`: File discovery functionality (`findAllMarkdownFiles`, `handleFindAllMarkdownFiles`)
- `read_handler.go`: File reading functionality (`handleReadMarkdownFile`, `findFirstFileByName`)

**Key Functions:**

- `findAllMarkdownFiles()` (`find.go`): Recursively discovers `.md` files in configured directories
- `handleFindAllMarkdownFiles()` (`find.go`): Resource handler returning JSON list of all markdown files with metadata
- `handleReadMarkdownFile()` (`read_handler.go`): Tool handler for reading individual file contents by filename only
- `findFirstFileByName()` (`read_handler.go`): Helper function that searches for files by name across all configured directories and returns the first match found

**Security Model:**

- Path validation prevents directory traversal attacks (`..` sequences blocked)
- File access restricted to configured directories only
- Only `.md` files are discovered and accessible
- Only filename-based search (no path traversal through tool interface)
- All file paths converted to absolute paths for validation

### MCP Interface

**Resource:** `markdown://find_all_files`

- Returns JSON array of discovered markdown files with metadata (path, name, relativePath)
- Includes directory list and file count

**Tools:**

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
- Isolated testing with temporary directories and mock data
- Comprehensive error handling and edge case coverage

## Key Dependencies

- `github.com/mark3labs/mcp-go v0.34.0`: MCP protocol implementation
- Standard library only (no external runtime dependencies)

## Usage Patterns

The server runs as a command-line tool requiring directory arguments:

```bash
./markdown-reader-mcp docs guides .     # Scan multiple directories
./markdown-reader-mcp /path/to/docs     # Scan specific directory
```

Designed to integrate with Claude Code via MCP configuration in `CLAUDE.md`:

```markdown
## Markdown Reader

- **Command**: `./markdown-reader-mcp`
- **Args**: `["docs", "guides", "."]`
```
