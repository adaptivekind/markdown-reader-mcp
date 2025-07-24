# Markdown Reader MCP Server

A Model Context Protocol (MCP) server built in Go with **read-only** access to
markdown documents from configured local directories. The scope of this MCP
server is explicitly constrained to read-only to minimise security concerns.

## Resources

### `markdown://list`

Returns a JSON list of all markdown files found in configured directories,
including file metadata.

## Tools

### `read_markdown_file`

Read the content of a specific markdown file by providing either:

- Full or relative file path (e.g., `docs/api.md`, `./guides/setup.md`)
- Just the filename (e.g., `README.md`, `api`, `setup`) - the server will search all configured directories

If multiple files with the same name exist, the first match found will be returned.

## Usage

Run the server with directories specified as arguments:

```bash
# Scan single directory
./markdown-reader-mcp ./my-path

# Scan multiple directories
./markdown-reader-mcp /path/to/docs /another/path ./local/docs

```

## Build the server

```bash
go build
```

## Integration with Claude Code

To use this MCP server with Claude Code, you need to configure it in your MCP settings.

### Configure Claude Code

Add the server to your Claude Code MCP configuration using one of these methods:

```bash
claude mcp add markdown-reader -- ./markdown-reader-mcp ~/my-markdown-directory
```

### Available Capabilities

Once configured, Claude Code will have access to:

**Resources:**

- `markdown://find_all_files` - Get a JSON list of all markdown files

**Tools:**

- `read_markdown_file` - Read content of specific markdown files

### Verify Configuration

In Claude code type `/mcp` to verify that the MCP server is registered and to view the capabilities.

### Example Usage

After setup, you can ask Claude Code to:

- "Show me all markdown files in the project"
- "Read the content of top-10-films"
- "Read the content of README.md"
- "Show me the api.md file"

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
