package main

import (
	"flag"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Config struct {
	Directories []string `json:"directories"`
}

var config Config

func main() {
	// Parse command line arguments
	flag.Parse()

	// Get directories from positional arguments, default to current directory
	args := flag.Args()
	if len(args) == 0 {
		config.Directories = []string{"."}
	} else {
		config.Directories = args
	}

	log.Printf("Scanning directories: %v", config.Directories)

	// Create MCP server
	s := server.NewMCPServer(
		"Markdown Reader",
		"0.0.1",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
	)

	// Add resource for listing markdown files
	s.AddResource(
		mcp.NewResource(
			"markdown://list",
			"List Markdown Files",
			mcp.WithResourceDescription("List of all markdown files in configured directories"),
			mcp.WithMIMEType("application/json"),
		),
		handleMarkdownList,
	)

	// Add tool for reading individual markdown files
	s.AddTool(
		mcp.NewTool("read_markdown_file",
			mcp.WithDescription("Read the content of a specific markdown file by path or filename"),
			mcp.WithString("file_path",
				mcp.Required(),
				mcp.Description("Path to the markdown file or just the filename (e.g., 'README.md' or 'docs/api.md')"),
			),
		),
		handleReadMarkdownFile,
	)

	// Start the server
	log.Println("Starting Markdown Reader MCP server...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
