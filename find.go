package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	DefaultPageSize    = 50
	DefaultMaxPageSize = 500
)

func handleFindMarkdownFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := extractQueryParam(req.Params.Arguments)
	pageSize := extractPageSizeParam(req.Params.Arguments)

	logger.Debug("find_markdown_files called", "query", query, "page_size", pageSize)

	files, err := findMarkdownFiles(query, pageSize)
	if err != nil {
		logger.Debug("find_markdown_files failed", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to find markdown files: %v", err)), nil
	}

	// Create file info objects with only filename (no absolute paths)
	fileInfos := make([]map[string]any, 0, len(files))
	for _, file := range files {
		fileInfos = append(fileInfos, map[string]any{
			"name": filepath.Base(file),
		})
	}

	result := map[string]any{
		"files": fileInfos,
		"count": len(fileInfos),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Debug("find_markdown_files failed to marshal JSON", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal file list: %v", err)), nil
	}

	logger.Debug("find_markdown_files completed successfully", "files_found", len(files))

	return mcp.NewToolResultText(string(jsonData)), nil
}

func shouldIgnoreDir(dirName string) bool {
	for _, pattern := range config.IgnoreDirs {
		matched, err := regexp.MatchString(pattern, dirName)
		if err != nil {
			logger.Debug("Invalid regex pattern", "pattern", pattern, "error", err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func findMarkdownFiles(query string, pageSize int) ([]string, error) {
	var allMarkdownFiles []string

	// Collect all markdown files from each directory
	for _, dir := range config.Directories {
		files := collectMarkdownFilesFromDir(dir)
		allMarkdownFiles = append(allMarkdownFiles, files...)
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
		pageSize = DefaultPageSize
	}

	if len(filteredFiles) <= pageSize {
		return filteredFiles, nil
	}

	return filteredFiles[:pageSize], nil
}

func extractQueryParam(arguments any) string {
	argsMap, ok := arguments.(map[string]any)
	if !ok {
		return ""
	}

	queryParam, exists := argsMap["query"]
	if !exists {
		return ""
	}

	queryStr, ok := queryParam.(string)
	if !ok {
		return ""
	}

	return queryStr
}

func extractPageSizeParam(arguments any) int {
	defaultPageSize := DefaultPageSize

	argsMap, ok := arguments.(map[string]any)
	if !ok {
		return defaultPageSize
	}

	pageSizeParam, exists := argsMap["page_size"]
	if !exists {
		return defaultPageSize
	}

	if pageSizeStr, ok := pageSizeParam.(string); ok {
		if parsedSize, err := strconv.Atoi(pageSizeStr); err == nil {
			return parsedSize
		}
	}

	if pageSizeFloat, ok := pageSizeParam.(float64); ok {
		return int(pageSizeFloat)
	}

	return defaultPageSize
}

func collectMarkdownFilesFromDir(dir string) []string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		logger.Warn("Could not resolve absolute path", "directory", dir, "error", err)
		return nil
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		logger.Warn("Directory does not exist", "directory", absDir)
		return nil
	}

	var files []string
	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() && shouldIgnoreDir(d.Name()) {
			return filepath.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		logger.Warn("Error walking directory", "directory", absDir, "error", err)
	}

	return files
}
