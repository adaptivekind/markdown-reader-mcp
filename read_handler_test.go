package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestFindFirstFileByName(t *testing.T) {
	// Setup test environment
	oldConfig := config
	oldLogger := logger
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	tempDir := "test/dir1"

	config = Config{Directories: []string{tempDir}}
	defer func() {
		config = oldConfig
		logger = oldLogger
	}()

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
			result, err := findFirstFileByName(tt.filename)

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

func TestHandleReadMarkdownFileResource(t *testing.T) {
	// Setup test environment
	oldConfig := config
	oldLogger := logger
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	testDir := "test/dir1"

	// Set config to test directory
	config = Config{Directories: []string{testDir}}
	defer func() {
		config = oldConfig
		logger = oldLogger
	}()

	tests := []struct {
		name        string
		filename    string
		wantError   bool
		wantContent string
	}{
		{
			name:        "read file in top level directory",
			filename:    "foo.md",
			wantError:   false,
			wantContent: "# Foo\n\nFoo markdown document\n",
		},
		{
			name:        "read file in child directory",
			filename:    "bar.md",
			wantError:   false,
			wantContent: "# Bar\n\nBar markdown document\n",
		},
		{
			name:        "read file without extension",
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
			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{
					URI: "markdown://" + tt.filename,
				},
			}

			result, err := handleReadMarkdownFileResource(context.Background(), req)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != 1 {
				t.Errorf("Expected 1 resource content, got %d", len(result))
				return
			}

			textResourceContent, ok := result[0].(mcp.TextResourceContents)
			if !ok {
				t.Errorf("Expected TextResourceContents, got %T", result[0])
				return
			}

			if textResourceContent.Text != tt.wantContent {
				t.Errorf("Expected content %q, got %q", tt.wantContent, textResourceContent.Text)
			}

			if textResourceContent.MIMEType != "text/markdown" {
				t.Errorf("Expected MIME type 'text/markdown', got %q", textResourceContent.MIMEType)
			}

			if textResourceContent.URI != req.Params.URI {
				t.Errorf("Expected URI %q, got %q", req.Params.URI, textResourceContent.URI)
			}
		})
	}
}
