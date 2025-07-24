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

**Server Setup (`main.go:24-74`)**

- Uses `github.com/mark3labs/mcp-go` library for MCP protocol implementation
- Accepts directory paths as positional command-line arguments (defaults to current directory)
- Registers one resource and three tools with the MCP server
- Communicates via stdio using JSON-RPC 2.0

**Key Functions:**

- `findMarkdownFiles()`: Recursively discovers `.md` files in configured directories
- `handleMarkdownList()`: Resource handler returning JSON list of all markdown files with metadata
- `handleReadMarkdownFile()`: Tool handler for reading individual file contents (supports path or filename search)
- `findFileByName()`: Helper function that searches for files by name across all configured directories

**Security Model:**

- Path validation prevents directory traversal attacks (`..` sequences blocked)
- File access restricted to configured directories only
- Only `.md` files are discovered and accessible
- All file paths converted to absolute paths for validation

### MCP Interface

**Resource:** `markdown://list`

- Returns JSON array of discovered markdown files with metadata (path, name, size, modified time)

**Tools:**

- `read_markdown_file`: Read content of specific file by full path, relative path, or just filename (searches all directories)
- `search_markdown`: Search text within markdown files (with optional directory filtering)
- `list_directories`: List configured scan directories

### Testing Architecture

**End-to-End Testing (`e2e_test.go`)**

- `MCPTestClient`: Custom test client that spawns server process and communicates via stdio
- Tests full MCP protocol handshake, resource/tool operations, and error handling
- Uses `test_data/` directory with sample markdown files for realistic testing
- Includes timeout handling and proper process cleanup

## Key Dependencies

- `github.com/mark3labs/mcp-go v0.34.0`: MCP protocol implementation
- Standard library only (no external runtime dependencies)

## Usage Patterns

The server runs as a command-line tool accepting directory arguments:

```bash
./markdown-reader-mcp                    # Scan current directory
./markdown-reader-mcp docs guides .     # Scan multiple directories
```

Designed to integrate with Claude Code via MCP configuration in `CLAUDE.md`:

```markdown
## Markdown Reader

- **Command**: `./markdown-reader-mcp`
- **Args**: `["docs", "guides", "."]`
```
