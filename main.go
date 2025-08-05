package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
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
	SSEMode      bool     `json:"sse_mode,omitempty"`
	SSEPort      int      `json:"sse_port,omitempty"`
	LogFile      string   `json:"log_file,omitempty"`
}

var (
	config     Config
	logger     *slog.Logger
	helpFlag   = flag.Bool("help", false, "Show usage information")
	debugFlag  = flag.Bool("debug", false, "Enable debug logging (overrides config)")
	quietFlag  = flag.Bool("quiet", false, "Disable debug logging (overrides config)")
	sseFlag    = flag.Bool("sse", false, "Enable SSE mode (overrides config)")
	stdoutFlag = flag.Bool("stdout", false, "Output logs to stdout (overrides log_file config)")
)

func showUsage() {
	fmt.Printf(`Markdown Reader MCP Server

A Model Context Protocol (MCP) server that provides read-only access to Markdown files
in configured directories. The server discovers and reads .md files only.

This server uses stdio transport and is designed to work with MCP clients like Claude.

USAGE:
  %s [options] [directories...]
  %s -help

OPTIONS:
  -help    Show this usage information
  -debug   Enable debug logging (overrides config file setting)
  -quiet   Disable debug logging (overrides config file setting)
  -sse     Enable SSE mode (overrides config file setting)
  -stdout  Output logs to stdout (overrides log_file config setting)

CONFIGURATION:
  The server can be configured in two ways:

  1. Command-line arguments (directories):
     %s ~/documents/notes ~/projects/docs /absolute/path

  2. Configuration file (recommended):
     Create ~/.config/markdown-reader-mcp/markdown-reader-mcp.json:
     {
       "directories": ["~/my/notes", "~/projects/docs", "."],
       "max_page_size": 100,
       "debug_logging": false,
       "ignore_dirs": ["\\.git$", "node_modules$", "vendor$"],
       "sse_mode": false,
       "sse_port": 8080,
       "log_file": "~/logs/markdown-reader-mcp.log"
     }

CONFIGURATION OPTIONS:
  directories    - Array of directory paths to scan for markdown files
  max_page_size  - Maximum results per page (default: %d)
  debug_logging  - Enable detailed debug logging (default: false)
  ignore_dirs    - Regex patterns for directories to ignore
                   (default: ["\\.git$", "node_modules$"])
  sse_mode       - Enable SSE transport mode (default: false)
  sse_port       - Port for SSE server (default: 8080)
  log_file       - Path to log file (default: stderr)

INTEGRATION:
  This server is designed to work with MCP clients like Claude Code:
    claude mcp add markdown-reader -- %s

CAPABILITIES PROVIDED:
  find_markdown_files  - Tool: Find markdown files with optional filtering and pagination
  markdown://files     - Resource: Read content of specific markdown file by filename

EXAMPLES:
  %s ~/documents/notes                    # Scan single directory
  %s ~/notes ~/docs .                     # Scan multiple directories
  %s                                      # Use config file
  %s -debug ~/docs                        # Enable debug logging via command line
  %s -quiet                               # Disable debug logging via command line
  %s -sse ~/docs                          # Enable SSE mode via command line
  %s -stdout ~/docs                       # Output logs to stdout via command line

For more information, see the README.md file.
`, os.Args[0], os.Args[0], os.Args[0], DefaultMaxPageSize, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

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

	// Handle help flag
	if *helpFlag {
		showUsage()
		os.Exit(0)
	}

	// Validate conflicting flags
	if *debugFlag && *quietFlag {
		fmt.Fprintf(os.Stderr, "Error: -debug and -quiet flags cannot be used together\n")
		os.Exit(1)
	}

	// Show debug logging status and source
	debugLogging := config.DebugLogging
	source := "config"
	if *debugFlag {
		debugLogging = true
		source = "command line"
	} else if *quietFlag {
		debugLogging = false
		source = "command line"
	}

	logLevel := slog.LevelWarn
	if debugLogging {
		logLevel = slog.LevelDebug
	}

	// Initialize basic logger for startup (will be reconfigured after loading config)
	logger = slog.New(newPrettyHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	logger.Debug("Debug logging is enabled", "source", source)

	// Get directories from positional arguments or config file
	args := flag.Args()
	if len(args) == 0 {
		// Try to load from config file
		cfg, err := loadConfigFromFile()
		if err != nil {
			logger.Error("No command arguments provided and could not load config file", "error", err)
			os.Exit(1)
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

	// Configure logger based on the loaded config
	configureLogger()

	logger.Info("Scanning directories", "directories", config.Directories)
	logger.Info("Ignoring directories matching patterns", "patterns", config.IgnoreDirs)

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

	// Add resource for reading individual markdown files
	s.AddResourceTemplate(
		mcp.NewResourceTemplate("markdown://{filename}", "Markdown Resource"),
		handleReadMarkdownFileResource,
	)

	// Determine SSE mode setting with command line flag taking precedence
	sseMode := config.SSEMode
	if *sseFlag {
		sseMode = true
	}

	// Start the server
	if sseMode {
		var port string
		if config.SSEPort != 0 {
			port = fmt.Sprintf("%d", config.SSEPort)
		} else if envPort := os.Getenv("PORT"); envPort != "" {
			port = envPort
		} else {
			port = "8080" // Default port
		}
		logger.Info("Starting Markdown Reader MCP server in SSE mode", "port", port)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(":" + port); err != nil {
			logger.Error("SSE server error", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Info("Starting Markdown Reader MCP server in stdio mode")
		if err := server.ServeStdio(s); err != nil {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}
}
