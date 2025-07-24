package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleReadMarkdownFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filename, err := req.RequireString("filename")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Security check: ensure the file path doesn't contain directory traversal
	if strings.Contains(filename, "..") {
		return mcp.NewToolResultError("invalid file path: directory traversal not allowed"), nil
	}

	var targetFile string

	// Check if this is just a filename (no path separators) - if so, search for it
	if !strings.Contains(filename, string(filepath.Separator)) {
		// Search for the file by name across all configured directories
		found, err := findFileByName(filename)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error searching for file: %v", err)), nil
		}
		if found == "" {
			return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", filename)), nil
		}
		targetFile = found
	} else {
		return mcp.NewToolResultError("filename looks like a path, it should be just the name of file"), nil
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

			if !d.IsDir() && strings.EqualFold(d.Name(), filename) {
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
