package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestFindMarkdownFiles(t *testing.T) {
	// Setup test environment
	oldConfig := config
	markdownDir := "test/dir1"

	tests := []struct {
		name      string
		dirs      []string
		wantCount int
		wantFiles []string
	}{
		{
			name:      "find all markdown files",
			dirs:      []string{markdownDir},
			wantCount: 4,
			wantFiles: []string{"README.md", "foo.md", "bar.md", "baz.md"},
		},
		{
			name:      "find files in non-existent directory",
			dirs:      []string{filepath.Join(markdownDir, "nonexistent")},
			wantCount: 0,
			wantFiles: []string{},
		},
		{
			name:      "find files in multiple directories",
			dirs:      []string{markdownDir, "test/dir2"},
			wantCount: 5,
			wantFiles: []string{"README.md", "foo.md", "bar.md", "baz.md", "cat.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = Config{Directories: tt.dirs}
			defer func() { config = oldConfig }()

			files, err := findAllMarkdownFiles()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(files) != tt.wantCount {
				t.Errorf("Expected %d files, got %d", tt.wantCount, len(files))
			}

			// Check that all expected files are found
			foundFiles := make(map[string]bool)
			for _, file := range files {
				basename := filepath.Base(file)
				foundFiles[basename] = true
			}

			for _, wantFile := range tt.wantFiles {
				if !foundFiles[wantFile] {
					t.Errorf("Expected to find file %s", wantFile)
				}
			}
		})
	}
}

func TestHandleFindAllMarkdown(t *testing.T) {
	// Setup test environment
	oldConfig := config
	testDir := "test/dir1"

	// Set config to test directory
	config = Config{Directories: []string{testDir}}
	defer func() { config = oldConfig }()

	tests := []struct {
		name      string
		req       mcp.ReadResourceRequest
		wantError bool
		wantFiles int
		wantDirs  []string
	}{
		{
			name: "successful list",
			req: mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: "markdown://find_all_files",
				},
			},
			wantError: false,
			wantFiles: 4,
			wantDirs:  []string{testDir},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleFindAllMarkdownFiles(context.Background(), tt.req)

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

			if len(result) != 1 {
				t.Errorf("Expected 1 resource content, got %d", len(result))
				return
			}

			textContent, ok := result[0].(mcp.TextResourceContents)
			if !ok {
				t.Errorf("Expected TextResourceContents, got %T", result[0])
				return
			}

			// Parse JSON response
			var listData map[string]any
			if err := json.Unmarshal([]byte(textContent.Text), &listData); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check directories
			dirs, ok := listData["directories"].([]any)
			if !ok {
				t.Error("Expected directories array in response")
				return
			}

			if len(dirs) != len(tt.wantDirs) {
				t.Errorf("Expected %d directories, got %d", len(tt.wantDirs), len(dirs))
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

				expectedFields := []string{"path", "name"}
				for _, field := range expectedFields {
					if _, exists := fileData[field]; !exists {
						t.Errorf("Expected field %s in file data", field)
					}
				}
			}
		})
	}
}
