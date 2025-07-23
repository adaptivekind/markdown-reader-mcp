package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
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

func main() {
	// Parse command line arguments
	var directoriesFlag = flag.String("dirs", ".", "Comma-separated list of directories to scan for markdown files")
	flag.Parse()

	// Parse directories from command line argument
	config.Directories = strings.Split(*directoriesFlag, ",")

	// Trim whitespace from each directory
	for i, dir := range config.Directories {
		config.Directories[i] = strings.TrimSpace(dir)
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
			"Markdown Files List",
			mcp.WithResourceDescription("List of all markdown files in configured directories"),
			mcp.WithMIMEType("application/json"),
		),
		handleMarkdownList,
	)

	// Add tool for reading individual markdown files
	s.AddTool(
		mcp.NewTool("read_markdown_file",
			mcp.WithDescription("Read the content of a specific markdown file"),
			mcp.WithString("file_path",
				mcp.Required(),
				mcp.Description("Path to the markdown file to read"),
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

func findMarkdownFiles() ([]string, error) {
	var markdownFiles []string

	for _, dir := range config.Directories {
		// Convert relative paths to absolute
		absDir, err := filepath.Abs(dir)
		if err != nil {
			log.Printf("Warning: Could not resolve absolute path for %s: %v", dir, err)
			continue
		}

		// Check if directory exists
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			log.Printf("Warning: Directory does not exist: %s", absDir)
			continue
		}

		err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip files that can't be accessed
			}

			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				markdownFiles = append(markdownFiles, path)
			}

			return nil
		})
		if err != nil {
			log.Printf("Warning: Error walking directory %s: %v", absDir, err)
		}
	}

	return markdownFiles, nil
}

func handleMarkdownList(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	files, err := findMarkdownFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find markdown files: %w", err)
	}

	// Create file info objects
	fileInfos := make([]map[string]interface{}, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue // Skip files that can't be accessed
		}

		fileInfos = append(fileInfos, map[string]interface{}{
			"path":         file,
			"name":         filepath.Base(file),
			"size":         info.Size(),
			"modified":     info.ModTime(),
			"relativePath": strings.TrimPrefix(file, filepath.Dir(file)+string(filepath.Separator)),
		})
	}

	result := map[string]interface{}{
		"directories": config.Directories,
		"files":       fileInfos,
		"count":       len(fileInfos),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal file list: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func handleReadMarkdownFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, err := req.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Security check: ensure the file path doesn't contain directory traversal
	if strings.Contains(filePath, "..") {
		return mcp.NewToolResultError("invalid file path: directory traversal not allowed"), nil
	}

	// Convert to absolute path if relative
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to resolve absolute path: %v", err)), nil
	}

	// Check if file exists and is a markdown file
	if !strings.HasSuffix(strings.ToLower(absPath), ".md") {
		return mcp.NewToolResultError(fmt.Sprintf("file is not a markdown file: %s", absPath)), nil
	}

	// Verify the file is within our configured directories
	isAllowed := false
	for _, dir := range config.Directories {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, absDir) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return mcp.NewToolResultError(fmt.Sprintf("file is not within configured directories: %s", absPath)), nil
	}

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file %s: %v", absPath, err)), nil
	}

	return mcp.NewToolResultText(string(content)), nil
}
