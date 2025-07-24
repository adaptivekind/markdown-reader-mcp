package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigWithMaxPageSize(t *testing.T) {
	// Create a temporary config file with max_page_size
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "markdown-reader-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "markdown-reader-mcp.json")

	// Test config data with max_page_size
	testConfig := Config{
		Directories: []string{"docs", "guides"},
		MaxPageSize: 100,
	}

	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test loading the config
	cfg, err := loadConfigFromFile()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.MaxPageSize != 100 {
		t.Errorf("Expected MaxPageSize 100, got %d", cfg.MaxPageSize)
	}
}

func TestConfigDefaultMaxPageSize(t *testing.T) {
	// Create a temporary config file without max_page_size
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "markdown-reader-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "markdown-reader-mcp.json")

	// Test config data without max_page_size
	testConfig := Config{
		Directories: []string{"docs", "guides"},
	}

	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test loading the config
	cfg, err := loadConfigFromFile()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.MaxPageSize != 500 {
		t.Errorf("Expected default MaxPageSize 500, got %d", cfg.MaxPageSize)
	}
}

func TestPaginationLimits(t *testing.T) {
	// Setup test environment
	oldConfig := config
	defer func() { config = oldConfig }()

	// Test with different max page sizes
	tests := []struct {
		name         string
		maxPageSize  int
		requestSize  int
		expectedSize int
	}{
		{
			name:         "request within limit",
			maxPageSize:  100,
			requestSize:  50,
			expectedSize: 50,
		},
		{
			name:         "request exceeds limit",
			maxPageSize:  100,
			requestSize:  150,
			expectedSize: 50, // Should fall back to default
		},
		{
			name:         "request zero uses default",
			maxPageSize:  100,
			requestSize:  0,
			expectedSize: 50, // Should use default
		},
		{
			name:         "request negative uses default",
			maxPageSize:  100,
			requestSize:  -10,
			expectedSize: 50, // Should use default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = Config{
				Directories: []string{"test/dir1"},
				MaxPageSize: tt.maxPageSize,
			}

			files, err := findMarkdownFiles("", tt.requestSize)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(files) > tt.expectedSize {
				t.Errorf("Expected at most %d files, got %d", tt.expectedSize, len(files))
			}
		})
	}
}
