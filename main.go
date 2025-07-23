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

	var targetFile string

	// Check if this is just a filename (no path separators) - if so, search for it
	if !strings.Contains(filePath, string(filepath.Separator)) && !strings.Contains(filePath, "/") {
		// Search for the file by name across all configured directories
		found, err := findFileByName(filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error searching for file: %v", err)), nil
		}
		if found == "" {
			return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", filePath)), nil
		}
		targetFile = found
	} else {
		// Handle as a full or relative path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to resolve absolute path: %v", err)), nil
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

		targetFile = absPath
	}

	// Check if file exists and is a markdown file
	if !strings.HasSuffix(strings.ToLower(targetFile), ".md") {
		return mcp.NewToolResultError(fmt.Sprintf("file is not a markdown file: %s", targetFile)), nil
	}

	// Read the file
	content, err := os.ReadFile(targetFile)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file %s: %v", targetFile, err)), nil
	}

	return mcp.NewToolResultText(string(content)), nil
}

// findFileByName searches for a markdown file by name across all configured directories
func findFileByName(filename string) (string, error) {
	// Ensure the filename has .md extension if not provided
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename = filename + ".md"
	}

	var matches []string

	for _, dir := range config.Directories {
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

			if !d.IsDir() && strings.ToLower(d.Name()) == strings.ToLower(filename) {
				matches = append(matches, path)
			}

			return nil
		})

		if err != nil {
			log.Printf("Warning: Error walking directory %s: %v", absDir, err)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("file not found: %s", filename)
	}

	if len(matches) > 1 {
		// Return the first match but log a warning about multiple matches
		log.Printf("Warning: Multiple files found with name %s, using first match: %s", filename, matches[0])
		for i, match := range matches {
			log.Printf("  Match %d: %s", i+1, match)
		}
	}

	return matches[0], nil
}
