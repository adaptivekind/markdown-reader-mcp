package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleReadMarkdownFileResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	logger.Debug("reading", "uri", req.Params.URI)

	// Extract filename from template parameters (file://{filename})
	filename := ""
	if req.Params.Arguments != nil {
		if filenameArg, ok := req.Params.Arguments["filename"].(string); ok {
			filename = filenameArg
		}
	}

	// Fallback: Extract from URI path for direct URI calls
	if filename == "" && strings.HasPrefix(req.Params.URI, "file://") {
		filename = strings.TrimPrefix(req.Params.URI, "file://")
	}

	if filename == "" {
		logger.Debug("read_markdown_file_resource missing filename parameter")
		return nil, fmt.Errorf("missing required parameter: filename")
	}

	logger.Debug("read_markdown_file_resource called", "filename", filename, "uri", req.Params.URI)

	// Security check: ensure the file path doesn't contain directory traversal
	if strings.Contains(filename, "..") {
		logger.Debug("read_markdown_file_resource blocked directory traversal attempt", "filename", filename)
		return nil, fmt.Errorf("invalid file path: directory traversal not allowed")
	}

	var targetFile string

	// Check if this is just a filename (no path separators) - if so, search for it
	if !strings.Contains(filename, string(filepath.Separator)) {
		// Search for the file by name across all configured directories
		found, err := findFirstFileByName(filename)
		if err != nil {
			logger.Debug("read_markdown_file_resource error searching for file", "error", err)
			return nil, fmt.Errorf("error searching for file: %v", err)
		}
		if found == "" {
			logger.Debug("read_markdown_file_resource file not found", "filename", filename)
			return nil, fmt.Errorf("file not found: %s", filename)
		}
		targetFile = found
		logger.Debug("read_markdown_file_resource found file", "file", targetFile)
	} else {
		logger.Debug("read_markdown_file_resource rejected path-like filename", "filename", filename)
		return nil, fmt.Errorf("filename looks like a path, it should be just the name of file")
	}

	// Check if file exists and is a markdown file
	if !strings.HasSuffix(strings.ToLower(targetFile), ".md") {
		logger.Debug("read_markdown_file_resource rejected non-markdown file", "file", targetFile)
		return nil, fmt.Errorf("file is not a markdown file: %s", targetFile)
	}

	// Read the file
	content, err := os.ReadFile(targetFile)
	if err != nil {
		logger.Debug("read_markdown_file_resource failed to read file", "error", err)
		return nil, fmt.Errorf("failed to read file %s: %v", targetFile, err)
	}

	logger.Debug("read_markdown_file_resource completed successfully", "bytes_read", len(content), "file", targetFile)

	// Create resource content
	resourceContent := mcp.TextResourceContents{
		URI:      req.Params.URI,
		MIMEType: "text/markdown",
		Text:     string(content),
	}

	return []mcp.ResourceContents{resourceContent}, nil
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
			logger.Warn("Could not resolve absolute path", "directory", dir, "error", err)
			continue
		}

		// Check if directory exists
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			logger.Warn("Directory does not exist", "directory", absDir)
			continue
		}

		var foundFile string
		err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip files that can't be accessed
			}

			// Skip directories that match ignore patterns
			if d.IsDir() && shouldIgnoreDir(d.Name()) {
				return filepath.SkipDir
			}

			if !d.IsDir() && strings.EqualFold(d.Name(), filename) {
				foundFile = path
				return filepath.SkipAll // Stop searching immediately after finding the first match
			}

			return nil
		})
		if err != nil {
			logger.Warn("Error walking directory", "directory", absDir, "error", err)
		}

		// Return immediately if we found a file in this directory
		if foundFile != "" {
			return foundFile, nil
		}
	}

	return "", fmt.Errorf("file not found: %s", filename)
}
