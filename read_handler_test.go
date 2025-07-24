package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestFindFileByName(t *testing.T) {
	// Setup test environment
	oldConfig := config
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"test.md":              "content1",
		"subdir/test.md":       "content2", // Duplicate name
		"subdir/unique.md":     "content3",
		"another/different.md": "content4",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	config = Config{Directories: []string{tempDir}}
	defer func() { config = oldConfig }()

	tests := []struct {
		name           string
		filename       string
		wantError      bool
		wantFound      bool
		expectMultiple bool
	}{
		{
			name:      "find existing unique file",
			filename:  "unique.md",
			wantError: false,
			wantFound: true,
		},
		{
			name:      "find existing file without extension",
			filename:  "unique",
			wantError: false,
			wantFound: true,
		},
		{
			name:           "find file with multiple matches",
			filename:       "test.md",
			wantError:      false,
			wantFound:      true,
			expectMultiple: true,
		},
		{
			name:      "find non-existent file",
			filename:  "nonexistent.md",
			wantError: true,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findFileByName(tt.filename)

			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.wantFound && result == "" {
				t.Error("Expected to find file but got empty result")
			}
			if !tt.wantFound && result != "" {
				t.Errorf("Expected empty result but got: %s", result)
			}

			if tt.wantFound && result != "" {
				// Verify the result is a valid path
				if !filepath.IsAbs(result) {
					t.Errorf("Expected absolute path, got: %s", result)
				}

				// Verify file exists
				if _, err := os.Stat(result); err != nil {
					t.Errorf("Found file does not exist: %s", result)
				}
			}
		})
	}
}

func TestHandleReadMarkdownFile(t *testing.T) {
	// Setup test environment
	oldConfig := config
	tempDir := t.TempDir()

	// Create test files
	testContent := "# Test File\nThis is test content."
	testFile := filepath.Join(tempDir, "test.md")
	nestedTestFile := filepath.Join(tempDir, "nested", "nested.md")

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(nestedTestFile), 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}
	if err := os.WriteFile(nestedTestFile, []byte("# Nested File\nNested content."), 0644); err != nil {
		t.Fatalf("Failed to create nested test file: %v", err)
	}

	// Set config to test directory
	config = Config{Directories: []string{tempDir}}
	defer func() { config = oldConfig }()

	tests := []struct {
		name        string
		filePath    string
		wantError   bool
		wantContent string
	}{
		{
			name:        "read existing file by full path",
			filePath:    testFile,
			wantError:   false,
			wantContent: testContent,
		},
		{
			name:        "read existing file by name only",
			filePath:    "test.md",
			wantError:   false,
			wantContent: testContent,
		},
		{
			name:        "read existing file by name without extension",
			filePath:    "test",
			wantError:   false,
			wantContent: testContent,
		},
		{
			name:      "read non-existent file",
			filePath:  "nonexistent.md",
			wantError: true,
		},
		{
			name:      "directory traversal attempt",
			filePath:  "../../../etc/passwd",
			wantError: true,
		},
		{
			name:      "read non-markdown file",
			filePath:  "test.txt",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "read_markdown_file",
					Arguments: map[string]any{
						"file_path": tt.filePath,
					},
				},
			}

			result, err := handleReadMarkdownFile(context.Background(), req)

			if tt.wantError {
				if err != nil {
					t.Errorf("Expected tool result error, got function error: %v", err)
					return
				}
				if !result.IsError {
					t.Error("Expected error result but got success")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.IsError {
				t.Errorf("Expected success but got error: %v", result.Content)
				return
			}

			if len(result.Content) != 1 {
				t.Errorf("Expected 1 content item, got %d", len(result.Content))
				return
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Errorf("Expected TextContent, got %T", result.Content[0])
				return
			}

			if textContent.Text != tt.wantContent {
				t.Errorf("Expected content %q, got %q", tt.wantContent, textContent.Text)
			}
		})
	}
}
