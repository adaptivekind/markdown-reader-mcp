package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "markdown-reader-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "markdown-reader-mcp.json")

	// Test config data
	testConfig := Config{
		Directories: []string{"docs", "guides", "."},
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

	if len(cfg.Directories) != 3 {
		t.Errorf("Expected 3 directories, got %d", len(cfg.Directories))
	}

	expected := []string{"docs", "guides", "."}
	for i, dir := range cfg.Directories {
		if dir != expected[i] {
			t.Errorf("Expected directory %s, got %s", expected[i], dir)
		}
	}
}

func TestLoadConfigFromFile_NotFound(t *testing.T) {
	// Mock the home directory to a non-existent path
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test loading non-existent config
	_, err := loadConfigFromFile()
	if err == nil {
		t.Error("Expected error when config file doesn't exist")
	}
}

func TestLoadConfigFromFile_InvalidJSON(t *testing.T) {
	// Create a temporary config file with invalid JSON
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "markdown-reader-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "markdown-reader-mcp.json")

	// Write invalid JSON
	invalidJSON := `{"directories": ["docs", "guides"` // Missing closing bracket and brace
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	// Mock the home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test loading the invalid config
	_, err := loadConfigFromFile()
	if err == nil {
		t.Error("Expected error when config file contains invalid JSON")
	}
}

func TestExpandTilde(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func() string
		wantErr  bool
	}{
		{
			name:     "no tilde",
			input:    "/absolute/path",
			expected: func() string { return "/absolute/path" },
			wantErr:  false,
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: func() string { return "relative/path" },
			wantErr:  false,
		},
		{
			name:  "tilde only",
			input: "~",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return home
			},
			wantErr: false,
		},
		{
			name:  "tilde with path",
			input: "~/Documents/projects",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "Documents/projects")
			},
			wantErr: false,
		},
		{
			name:     "tilde in middle (not expanded)",
			input:    "/path/~/file",
			expected: func() string { return "/path/~/file" },
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandTilde(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandTilde() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				expected := tt.expected()
				if result != expected {
					t.Errorf("expandTilde() = %v, want %v", result, expected)
				}
			}
		})
	}
}

func TestLoadConfigFromFileWithTilde(t *testing.T) {
	// Create a temporary config file with tilde paths
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "markdown-reader-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "markdown-reader-mcp.json")

	// Test config data with tilde paths
	testConfig := Config{
		Directories: []string{"~/Documents", "~/Desktop/notes", "/absolute/path"},
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

	if len(cfg.Directories) != 3 {
		t.Errorf("Expected 3 directories, got %d", len(cfg.Directories))
	}

	// Check that tilde paths were expanded
	expectedDirs := []string{
		filepath.Join(tempDir, "Documents"),
		filepath.Join(tempDir, "Desktop/notes"),
		"/absolute/path",
	}

	for i, dir := range cfg.Directories {
		if dir != expectedDirs[i] {
			t.Errorf("Expected directory %s, got %s", expectedDirs[i], dir)
		}

		// Verify tilde was expanded (should not contain ~ anymore except for absolute paths)
		if strings.Contains(dir, "~") && !strings.HasPrefix(expectedDirs[i], "/") {
			t.Errorf("Tilde was not expanded in directory: %s", dir)
		}
	}
}
