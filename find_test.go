package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestFindMarkdownFiles(t *testing.T) {
	// Setup test environment
	oldConfig := config
	oldLogger := logger
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	defer func() {
		config = oldConfig
		logger = oldLogger
	}()
	markdownDir := "test/dir1"

	tests := []struct {
		name      string
		dirs      []string
		query     string
		pageSize  int
		wantCount int
		wantFiles []string
	}{
		{
			name:      "find all markdown files",
			dirs:      []string{markdownDir},
			query:     "",
			pageSize:  0,
			wantCount: 4,
			wantFiles: []string{"README.md", "foo.md", "bar.md", "baz.md"},
		},
		{
			name:      "find files in non-existent directory",
			dirs:      []string{filepath.Join(markdownDir, "nonexistent")},
			query:     "",
			pageSize:  0,
			wantCount: 0,
			wantFiles: []string{},
		},
		{
			name:      "find files in multiple directories",
			dirs:      []string{markdownDir, "test/dir2"},
			query:     "",
			pageSize:  0,
			wantCount: 5,
			wantFiles: []string{"README.md", "foo.md", "bar.md", "baz.md", "cat.md"},
		},
		{
			name:      "find files with query filter",
			dirs:      []string{markdownDir},
			query:     "foo",
			pageSize:  0,
			wantCount: 1,
			wantFiles: []string{"foo.md"},
		},
		{
			name:      "find files with pagination",
			dirs:      []string{markdownDir},
			query:     "",
			pageSize:  2,
			wantCount: 2,
			wantFiles: []string{"README.md", "foo.md", "bar.md", "baz.md"}, // Any 2 from these files
		},
		{
			name:      "find files with query and pagination",
			dirs:      []string{markdownDir, "test/dir2"},
			query:     "a",
			pageSize:  1,
			wantCount: 1,
			wantFiles: []string{"README.md", "bar.md", "baz.md", "cat.md"}, // Any 1 file containing 'a'
		},
		{
			name:      "ignore directories matching regex patterns",
			dirs:      []string{"test/ignore_test"},
			query:     "",
			pageSize:  0,
			wantCount: 2,
			wantFiles: []string{"README.md", "main.md"}, // Should ignore .git and node_modules
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = Config{
				Directories: tt.dirs,
				MaxPageSize: DefaultMaxPageSize,
				IgnoreDirs:  []string{`\.git$`, `node_modules$`}, // Default ignore patterns
			}

			files, err := findMarkdownFiles(tt.query, tt.pageSize)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(files) != tt.wantCount {
				t.Errorf("Expected %d files, got %d", tt.wantCount, len(files))
			}

			// Check that expected files are found (allowing for different order or pagination)
			foundFiles := make(map[string]bool)
			for _, file := range files {
				basename := filepath.Base(file)
				foundFiles[basename] = true
			}

			// For pagination tests, just check that we have the expected count
			if tt.pageSize > 0 {
				// For pagination, we just verify the count and that all found files are valid
				for foundFile := range foundFiles {
					isValidFile := slices.Contains(tt.wantFiles, foundFile)
					if !isValidFile {
						t.Errorf("Found unexpected file %s", foundFile)
					}
				}
			} else {
				// For non-pagination tests, check exact matches
				for _, wantFile := range tt.wantFiles {
					if !foundFiles[wantFile] {
						t.Errorf("Expected to find file %s", wantFile)
					}
				}
			}
		})
	}
}

func TestShouldIgnoreDir(t *testing.T) {
	// Setup test environment
	oldConfig := config
	config = Config{
		Directories:  []string{},
		MaxPageSize:  DefaultMaxPageSize,
		DebugLogging: false,
		IgnoreDirs:   []string{`^\.git$`, `^node_modules$`, `^temp.+$`},
	}
	defer func() { config = oldConfig }()

	tests := []struct {
		dirName      string
		shouldIgnore bool
	}{
		{".git", true},
		{".gitignore", false}, // Should not match because it doesn't exactly match .git
		{"node_modules", true},
		{"my_node_modules", false}, // Should not match because it doesn't start with node_modules
		{"temp", false},            // Should not match because temp.+ requires at least one character after temp
		{"temp123", true},
		{"tempdir", true},
		{"src", false},
		{"docs", false},
	}

	for _, tt := range tests {
		t.Run(tt.dirName, func(t *testing.T) {
			result := shouldIgnoreDir(tt.dirName)
			if result != tt.shouldIgnore {
				t.Errorf("shouldIgnoreDir(%q) = %v, want %v", tt.dirName, result, tt.shouldIgnore)
			}
		})
	}
}

func TestHandleFindAllMarkdown(t *testing.T) {
	// Setup test environment
	oldConfig := config
	oldLogger := logger
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	testDir := "test/dir1"

	// Set config to test directory
	config = Config{Directories: []string{testDir}, MaxPageSize: DefaultMaxPageSize}
	defer func() {
		config = oldConfig
		logger = oldLogger
	}()

	tests := []struct {
		name      string
		req       mcp.CallToolRequest
		wantError bool
		wantFiles int
		wantDirs  []string
	}{
		{
			name: "successful list",
			req: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "find_markdown_files",
					Arguments: map[string]any{},
				},
			},
			wantError: false,
			wantFiles: 4,
			wantDirs:  []string{testDir},
		},
		{
			name: "list with query",
			req: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "find_markdown_files",
					Arguments: map[string]any{
						"query": "foo",
					},
				},
			},
			wantError: false,
			wantFiles: 1,
			wantDirs:  []string{testDir},
		},
		{
			name: "list with page size",
			req: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "find_markdown_files",
					Arguments: map[string]any{
						"page_size": "2",
					},
				},
			},
			wantError: false,
			wantFiles: 2,
			wantDirs:  []string{testDir},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleFindMarkdownFiles(context.Background(), tt.req)

			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if tt.wantError {
				return
			}

			if result == nil {
				t.Error("Expected tool result, got nil")
				return
			}

			if result.IsError {
				t.Errorf("Tool returned error: %v", result.Content)
				return
			}

			if len(result.Content) == 0 {
				t.Error("Expected content in tool result")
				return
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Errorf("Expected TextContent, got %T", result.Content[0])
				return
			}

			text := textContent.Text

			var listData map[string]any
			if err := json.Unmarshal([]byte(text), &listData); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check files count
			files, ok := listData["files"].([]any)
			if !ok {
				t.Error("Expected files array in response")
				return
			}

			if len(files) != tt.wantFiles {
				t.Errorf("Expected %d files, got %d", tt.wantFiles, len(files))
			}

			// Check count field
			count, ok := listData["count"].(float64)
			if !ok {
				t.Error("Expected count field in response")
				return
			}

			if int(count) != tt.wantFiles {
				t.Errorf("Expected count %d, got %d", tt.wantFiles, int(count))
			}

			// Verify file structure
			for _, file := range files {
				fileData, ok := file.(map[string]any)
				if !ok {
					t.Error("Expected file to be an object")
					continue
				}

				expectedFields := []string{"name"}
				for _, field := range expectedFields {
					if _, exists := fileData[field]; !exists {
						t.Errorf("Expected field %s in file data", field)
					}
				}
			}
		})
	}
}

func TestHandleFindMarkdownFilesWithIgnoredDirs(t *testing.T) {
	// Setup test environment with ignore test directory
	oldConfig := config
	oldLogger := logger
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	testDir := "test/ignore_test"
	config = Config{
		Directories:  []string{testDir},
		MaxPageSize:  DefaultMaxPageSize,
		DebugLogging: false,
		IgnoreDirs:   []string{`\.git$`, `node_modules$`},
	}
	defer func() {
		config = oldConfig
		logger = oldLogger
	}()

	tests := []struct {
		name      string
		req       mcp.CallToolRequest
		wantError bool
		wantFiles int
		wantDirs  []string
	}{
		{
			name: "ignore .git and node_modules directories",
			req: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "find_markdown_files",
					Arguments: map[string]any{},
				},
			},
			wantError: false,
			wantFiles: 2, // Should only find README.md and src/main.md, ignoring .git/config.md and node_modules/package.md
			wantDirs:  []string{testDir},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleFindMarkdownFiles(context.Background(), tt.req)

			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if tt.wantError {
				return
			}

			if result == nil {
				t.Error("Expected tool result, got nil")
				return
			}

			if result.IsError {
				t.Errorf("Tool returned error: %v", result.Content)
				return
			}

			if len(result.Content) == 0 {
				t.Error("Expected content in tool result")
				return
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Errorf("Expected TextContent, got %T", result.Content[0])
				return
			}

			text := textContent.Text

			// Parse JSON response
			var listData map[string]any
			if err := json.Unmarshal([]byte(text), &listData); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check files count
			files, ok := listData["files"].([]any)
			if !ok {
				t.Error("Expected files array in response")
				return
			}

			if len(files) != tt.wantFiles {
				t.Errorf("Expected %d files, got %d", tt.wantFiles, len(files))
			}

			// Verify that files are returned (can't verify path exclusion since paths are not exposed for security)
			if len(files) == 0 {
				t.Error("Expected to find some files, but got none")
			}
		})
	}
}
