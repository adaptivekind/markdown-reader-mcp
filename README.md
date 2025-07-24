# Markdown Reader MCP Server

A Model Context Protocol (MCP) server built in Go with **read-only** access to
markdown documents from configured local directories. The scope of this MCP
server is explicitly constrained to read-only to minimise security concerns.

The server includes debug logging that tracks each tool call with input parameters, execution time, and results.

## Tools

### `find_markdown_files`

Find markdown files in configured directories with optional filtering and pagination.

**Parameters:**
- `query` (optional): Filter files by name containing this string. If not provided, matches all files.
- `page_size` (optional): Limit number of results returned. Default is 50, configurable maximum in config file (default max: 500).

Returns a JSON list of matching markdown files with metadata (path, name, relativePath) along with directory list and file count.

### `read_markdown_file`

Read the content of a specific markdown file by providing just the filename:

- Filename with extension (e.g., `README.md`, `api.md`)
- Filename without extension (e.g., `README`, `api`) - `.md` extension will be added automatically

The server will search all configured directories and return the first match found.
Path traversal (e.g., `../`, `docs/api.md`) is not supported for security reasons.

## Usage

The server can be configured in two ways:

### Command-line Arguments

Run the server with directories specified as arguments:

```sh
# Scan single directory
./markdown-reader-mcp ./my-path

# Scan multiple directories
./markdown-reader-mcp /path/to/docs /another/path ./local/docs
```

### Configuration File

Create a configuration file at `~/.config/markdown-reader-mcp/markdown-reader-mcp.json`:

```json
{
  "directories": ["~/Documents/notes", "~/projects/docs", "/absolute/path"],
  "max_page_size": 100,
  "debug_logging": true
}
```

Then run without arguments:

```sh
./markdown-reader-mcp
```

**Configuration Options:**
- `directories`: Array of directory paths to scan for markdown files
- `max_page_size` (optional): Maximum number of results that can be returned in a single page. Defaults to 500 if not specified.
- `debug_logging` (optional): Enable detailed debug logging for each tool call. Defaults to false if not specified.

**Note:**
- Command-line arguments take precedence over the configuration file. If both are provided, the command-line arguments will be used.
- Tilde (`~`) in directory paths will be automatically expanded to your home directory.

## Build the server

```sh
go build
```

## Install

```sh
go install
```

## Integration with Claude Code

To use this MCP server with Claude Code, you need to configure it in your MCP settings.

### Configure Claude Code

Add the server to your Claude Code MCP configuration using one of these methods:

```sh
claude mcp add markdown-reader -- markdown-reader-mcp ~/my-markdown-directory
```

Or, if using the configuration file approach:

```sh
claude mcp add markdown-reader -- markdown-reader-mcp
```

Add server for all your projects on this machine

```sh
claude mcp add markdown-reader -s user -- markdown-reader-mcp ~/my-markdown-directory
```

Or with configuration file:

```sh
claude mcp add markdown-reader -s user -- markdown-reader-mcp
```

List MCP servers installed

```sh
claude mcp list
```

Remove MCP server

```sh
claude mcp remove markdown-reader
```

### Available Capabilities

Once configured, Claude Code will have access to:

**Tools:**

- `find_markdown_files` - Find markdown files with optional query filtering and pagination
- `read_markdown_file` - Read content of specific markdown files

### Verify Configuration

In Claude code type `/mcp` to verify that the MCP server is registered and to view the capabilities.

### Example Usage

After setup, you can ask Claude Code to:

- "Show me all markdown files in the project" (uses the `find_markdown_files` tool)
- "Find files containing 'api' in the name" (uses the `find_markdown_files` tool with query parameter)
- "Show me the first 10 markdown files" (uses the `find_markdown_files` tool with page_size parameter)
- "Read the content of README" (uses `read_markdown_file` tool with filename)
- "Read the content of README.md" (uses `read_markdown_file` tool with filename)
- "Show me the api file" (uses `read_markdown_file` tool, searches for `api.md`)

## Debug Logging

When enabled via the `debug_logging` configuration option, the server logs detailed debug information for each tool call, including:

- Input parameters (query, page_size, filename)
- Execution time for each operation
- Number of results found or bytes read
- Error conditions and security blocks

Debug logs are prefixed with `[DEBUG]` and include timing information to help with performance monitoring. Debug logging is **disabled by default** for performance and to reduce log noise.

## Development Setup

1. Clone the repository
2. Install pre-commit framework:

```sh
pip install pre-commit
```

3. Install the pre-commit hooks:

```sh
pre-commit install
```
