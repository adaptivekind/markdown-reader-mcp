package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func handleFindMarkdownFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse query parameter
	query := ""
	if req.Params.Arguments != nil {
		if argsMap, ok := req.Params.Arguments.(map[string]any); ok {
			if queryParam, exists := argsMap["query"]; exists {
				if queryStr, ok := queryParam.(string); ok {
					query = queryStr
				}
			}
		}
	}

	// Parse page_size parameter
	pageSize := 50 // Default page size
	if req.Params.Arguments != nil {
		if argsMap, ok := req.Params.Arguments.(map[string]any); ok {
			if pageSizeParam, exists := argsMap["page_size"]; exists {
				if pageSizeStr, ok := pageSizeParam.(string); ok {
					if parsedSize, err := strconv.Atoi(pageSizeStr); err == nil {
						pageSize = parsedSize
					}
				} else if pageSizeFloat, ok := pageSizeParam.(float64); ok {
					pageSize = int(pageSizeFloat)
				}
			}
		}
	}

	files, err := findMarkdownFiles(query, pageSize)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find markdown files: %v", err)), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal file list: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func findMarkdownFiles(query string, pageSize int) ([]string, error) {
	var allMarkdownFiles []string

	// Collect all markdown files first
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
				allMarkdownFiles = append(allMarkdownFiles, path)
			}

			return nil
		})
		if err != nil {
			log.Printf("Warning: Error walking directory %s: %v", absDir, err)
		}
	}

	// Filter by query if provided
	var filteredFiles []string
	if query != "" {
		queryLower := strings.ToLower(query)
		for _, file := range allMarkdownFiles {
			filename := strings.ToLower(filepath.Base(file))
			if strings.Contains(filename, queryLower) {
				filteredFiles = append(filteredFiles, file)
			}
		}
	} else {
		filteredFiles = allMarkdownFiles
	}

	// Apply pagination
	if pageSize <= 0 || pageSize > config.MaxPageSize {
		pageSize = 50 // Default page size
	}

	if len(filteredFiles) <= pageSize {
		return filteredFiles, nil
	}

	return filteredFiles[:pageSize], nil
}
