package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

const DefaultPageSize = 50

func handleFindMarkdownFiles(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	startTime := time.Now()

	query := extractQueryParam(req.Params.Arguments)
	pageSize := extractPageSizeParam(req.Params.Arguments)

	if config.DebugLogging {
		log.Printf("[DEBUG] find_markdown_files called with query='%s', page_size=%d", query, pageSize)
	}

	files, err := findMarkdownFiles(query, pageSize)
	if err != nil {
		if config.DebugLogging {
			duration := time.Since(startTime)
			log.Printf("[DEBUG] find_markdown_files failed after %v: %v", duration, err)
		}
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
		if config.DebugLogging {
			duration := time.Since(startTime)
			log.Printf("[DEBUG] find_markdown_files failed to marshal JSON after %v: %v", duration, err)
		}
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal file list: %v", err)), nil
	}

	if config.DebugLogging {
		duration := time.Since(startTime)
		log.Printf("[DEBUG] find_markdown_files completed successfully in %v, found %d files", duration, len(files))
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func shouldIgnoreDir(dirName string) bool {
	for _, pattern := range config.IgnoreDirs {
		matched, err := regexp.MatchString(pattern, dirName)
		if err != nil {
			// If regex is invalid, log warning and continue
			if config.DebugLogging {
				log.Printf("[DEBUG] Invalid regex pattern '%s': %v", pattern, err)
			}
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
		log.Printf("Warning: Could not resolve absolute path for %s: %v", dir, err)
		return nil
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Printf("Warning: Directory does not exist: %s", absDir)
		return nil
	}

	var files []string
	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files that can't be accessed
		}

		if d.IsDir() && shouldIgnoreDir(d.Name()) {
			if config.DebugLogging {
				log.Printf("[DEBUG] Ignoring directory: %s", path)
			}
			return filepath.SkipDir
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		log.Printf("Warning: Error walking directory %s: %v", absDir, err)
	}

	return files
}
