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
	tempDir := "test/dir1"

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
			name:      "find existing file",
			filename:  "foo.md",
			wantError: false,
			wantFound: true,
		},
		{
			name:      "find existing file without extension",
			filename:  "foo",
			wantError: false,
			wantFound: true,
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
	testDir := "test/dir1"

	// Set config to test directory
	config = Config{Directories: []string{testDir}}
	defer func() { config = oldConfig }()

	tests := []struct {
		name        string
		filename    string
		wantError   bool
		wantContent string
	}{
		{
			name:        "read existing file by name only",
			filename:    "foo.md",
			wantError:   false,
			wantContent: "# Foo\n\nFoo markdown document\n",
		},
		{
			name:        "read existing file by name without extension",
			filename:    "foo",
			wantError:   false,
			wantContent: "# Foo\n\nFoo markdown document\n",
		},
		{
			name:      "read non-existent file",
			filename:  "nonexistent.md",
			wantError: true,
		},
		{
			name:      "directory traversal attempt",
			filename:  "../../../etc/passwd",
			wantError: true,
		},
		{
			name:      "read non-markdown file",
			filename:  "test.txt",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "read_markdown_file",
					Arguments: map[string]any{
						"filename": tt.filename,
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
