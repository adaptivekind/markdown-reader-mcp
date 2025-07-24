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
	flag.Parse()

	// Get directories from positional arguments
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("Markdown directory (or directories) must by provided as command arguments")
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
			"markdown://find_all",
			"Find all Markdown Files",
			mcp.WithResourceDescription("Find all known markdown files"),
			mcp.WithMIMEType("application/json"),
		),
		handleFindMarkdown,
	)

	// Add tool for reading individual markdown files
	s.AddTool(
		mcp.NewTool("read_markdown_file",
			mcp.WithDescription("Read the contents of a specific markdown file by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the markdown file (e.g., 'README' or 'README.md')"),
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
