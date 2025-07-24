package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleFindAllMarkdownFiles(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	files, err := findAllMarkdownFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to find markdown files: %w", err)
	}

	// Create file info objects
	fileInfos := make([]map[string]any, 0, len(files))
	for _, file := range files {
		fileInfos = append(fileInfos, map[string]any{
			"path":         file,
			"name":         filepath.Base(file),
			"relativePath": strings.TrimPrefix(file, filepath.Dir(file)+string(filepath.Separator)),
		})
	}

	result := map[string]any{
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

func findAllMarkdownFiles() ([]string, error) {
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
