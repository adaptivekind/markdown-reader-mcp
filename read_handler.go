package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleReadMarkdownFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	filename, err := req.RequireString("filename")
	if err != nil {
		if config.DebugLogging {
			duration := time.Since(startTime)
			log.Printf("[DEBUG] read_markdown_file failed to parse filename after %v: %v", duration, err)
		}
		return mcp.NewToolResultError(err.Error()), nil
	}

	if config.DebugLogging {
		log.Printf("[DEBUG] read_markdown_file called with filename='%s'", filename)
	}

	// Security check: ensure the file path doesn't contain directory traversal
	if strings.Contains(filename, "..") {
		if config.DebugLogging {
			duration := time.Since(startTime)
			log.Printf("[DEBUG] read_markdown_file blocked directory traversal attempt after %v: filename='%s'", duration, filename)
		}
		return mcp.NewToolResultError("invalid file path: directory traversal not allowed"), nil
	}

	var targetFile string

	// Check if this is just a filename (no path separators) - if so, search for it
	if !strings.Contains(filename, string(filepath.Separator)) {
		// Search for the file by name across all configured directories
		found, err := findFirstFileByName(filename)
		if err != nil {
			if config.DebugLogging {
				duration := time.Since(startTime)
				log.Printf("[DEBUG] read_markdown_file error searching for file after %v: %v", duration, err)
			}
			return mcp.NewToolResultError(fmt.Sprintf("error searching for file: %v", err)), nil
		}
		if found == "" {
			if config.DebugLogging {
				duration := time.Since(startTime)
				log.Printf("[DEBUG] read_markdown_file file not found after %v: filename='%s'", duration, filename)
			}
			return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", filename)), nil
		}
		targetFile = found
		if config.DebugLogging {
			log.Printf("[DEBUG] read_markdown_file found file: %s", targetFile)
		}
	} else {
		if config.DebugLogging {
			duration := time.Since(startTime)
			log.Printf("[DEBUG] read_markdown_file rejected path-like filename after %v: filename='%s'", duration, filename)
		}
		return mcp.NewToolResultError("filename looks like a path, it should be just the name of file"), nil
	}

	// Check if file exists and is a markdown file
	if !strings.HasSuffix(strings.ToLower(targetFile), ".md") {
		if config.DebugLogging {
			duration := time.Since(startTime)
			log.Printf("[DEBUG] read_markdown_file rejected non-markdown file after %v: file='%s'", duration, targetFile)
		}
		return mcp.NewToolResultError(fmt.Sprintf("file is not a markdown file: %s", targetFile)), nil
	}

	// Read the file
	content, err := os.ReadFile(targetFile)
	if err != nil {
		if config.DebugLogging {
			duration := time.Since(startTime)
			log.Printf("[DEBUG] read_markdown_file failed to read file after %v: %v", duration, err)
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file %s: %v", targetFile, err)), nil
	}

	if config.DebugLogging {
		duration := time.Since(startTime)
		log.Printf("[DEBUG] read_markdown_file completed successfully in %v, read %d bytes from '%s'", duration, len(content), targetFile)
	}

	return mcp.NewToolResultText(string(content)), nil
}

// findFirstFileByName searches for a markdown file by name across all configured directories
// and returns the first match found
func findFirstFileByName(filename string) (string, error) {
	// Ensure the filename has .md extension if not provided
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename = filename + ".md"
	}

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

		var foundFile string
		err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip files that can't be accessed
			}

			if !d.IsDir() && strings.EqualFold(d.Name(), filename) {
				foundFile = path
				return filepath.SkipAll // Stop searching immediately after finding the first match
			}

			return nil
		})

		if err != nil {
			log.Printf("Warning: Error walking directory %s: %v", absDir, err)
		}

		// Return immediately if we found a file in this directory
		if foundFile != "" {
			return foundFile, nil
		}
	}

	return "", fmt.Errorf("file not found: %s", filename)
}
