package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Config struct {
	Directories []string `json:"directories"`
}

var config Config

func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}

func loadConfigFromFile() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "markdown-reader-mcp", "markdown-reader-mcp.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Expand tilde in directory paths
	for i, dir := range cfg.Directories {
		expandedDir, err := expandTilde(dir)
		if err != nil {
			return nil, err
		}
		cfg.Directories[i] = expandedDir
	}

	return &cfg, nil
}

func main() {
	flag.Parse()

	// Get directories from positional arguments or config file
	args := flag.Args()
	if len(args) == 0 {
		// Try to load from config file
		cfg, err := loadConfigFromFile()
		if err != nil {
			log.Fatalf("No command arguments provided and could not load config file: %v", err)
		}
		config = *cfg
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
			"markdown://find_all_files",
			"Find all Markdown Files",
			mcp.WithResourceDescription("Find all known markdown files"),
			mcp.WithMIMEType("application/json"),
		),
		handleFindAllMarkdownFiles,
	)

	// Add tool for reading individual markdown files
	s.AddTool(
		mcp.NewTool("read_markdown_file",
			mcp.WithDescription("Read the contents of a specific markdown file by name"),
			mcp.WithString("filename",
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
