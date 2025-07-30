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
	Directories  []string `json:"directories"`
	MaxPageSize  int      `json:"max_page_size,omitempty"`
	DebugLogging bool     `json:"debug_logging,omitempty"`
	IgnoreDirs   []string `json:"ignore_dirs,omitempty"`
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

	if cfg.MaxPageSize == 0 {
		cfg.MaxPageSize = DefaultMaxPageSize
	}

	if len(cfg.IgnoreDirs) == 0 {
		cfg.IgnoreDirs = []string{`\.git$`, `node_modules$`}
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
		// Set default max page size for command-line usage
		config.MaxPageSize = DefaultMaxPageSize
		// Debug logging is disabled by default for command-line usage
		config.DebugLogging = false
		// Set default ignore directories for command-line usage
		config.IgnoreDirs = []string{`\.git$`, `node_modules$`}
	}

	log.Printf("Scanning directories: %v", config.Directories)
	log.Printf("Ignoring directories matching patterns: %v", config.IgnoreDirs)

	if config.DebugLogging {
		log.Printf("[CONFIG] Debug logging is enabled")
	}

	// Create MCP server
	s := server.NewMCPServer(
		"Markdown Reader",
		"0.0.1",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
	)

	// Add tool for finding markdown files
	s.AddTool(
		mcp.NewTool("find_markdown_files",
			mcp.WithDescription("Find all markdown files in configured directories"),
			mcp.WithString("query",
				mcp.Description("Query to find matching files. If not set, then it matches all files. If a string is sent then files containing that text is returned."),
			),
			mcp.WithString("page_size",
				mcp.Description("Number of results in a page"),
			),
		),
		handleFindMarkdownFiles,
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
