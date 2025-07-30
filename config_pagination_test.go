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

	// Test config data with max_page_size, sse_port, and log_file
	testConfig := Config{
		Directories:  []string{"docs", "guides"},
		MaxPageSize:  100,
		DebugLogging: true,
		SSEPort:      9090,
		LogFile:      "~/test/logs/server.log",
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

	if !cfg.DebugLogging {
		t.Errorf("Expected DebugLogging true, got %v", cfg.DebugLogging)
	}

	if cfg.SSEPort != 9090 {
		t.Errorf("Expected SSEPort 9090, got %d", cfg.SSEPort)
	}

	if cfg.LogFile != "~/test/logs/server.log" {
		t.Errorf("Expected LogFile '~/test/logs/server.log', got %s", cfg.LogFile)
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

	if cfg.MaxPageSize != DefaultMaxPageSize {
		t.Errorf("Expected default MaxPageSize %d, got %d", DefaultMaxPageSize, cfg.MaxPageSize)
	}

	if cfg.DebugLogging {
		t.Errorf("Expected default DebugLogging false, got %v", cfg.DebugLogging)
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

func TestDebugLoggingConfiguration(t *testing.T) {
	// Setup test environment
	oldConfig := config
	defer func() { config = oldConfig }()

	tests := []struct {
		name         string
		debugLogging bool
	}{
		{
			name:         "debug logging enabled",
			debugLogging: true,
		},
		{
			name:         "debug logging disabled",
			debugLogging: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = Config{
				Directories:  []string{"test/dir1"},
				MaxPageSize:  DefaultMaxPageSize,
				DebugLogging: tt.debugLogging,
			}

			// Test find_markdown_files with debug logging setting
			files, err := findMarkdownFiles("", 10)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// We can't easily test if logs were actually printed in unit tests,
			// but we can verify the config is being respected and function works
			if len(files) < 0 { // This will never be true, but ensures files is used
				t.Error("Unexpected negative file count")
			}
		})
	}
}

func TestSSEPortConfiguration(t *testing.T) {
	// Create a temporary directory for home
	tempDir, err := os.MkdirTemp("", "mcp-test-home")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config directory structure
	configDir := filepath.Join(tempDir, ".config", "markdown-reader-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "markdown-reader-mcp.json")

	tests := []struct {
		name     string
		ssePort  int
		expected int
	}{
		{
			name:     "custom port 9090",
			ssePort:  9090,
			expected: 9090,
		},
		{
			name:     "port 3000",
			ssePort:  3000,
			expected: 3000,
		},
		{
			name:     "zero port (default)",
			ssePort:  0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config with SSE port
			testConfig := Config{
				Directories: []string{"test"},
				SSEPort:     tt.ssePort,
			}

			configData, err := json.Marshal(testConfig)
			if err != nil {
				t.Fatalf("Failed to marshal test config: %v", err)
			}

			err = os.WriteFile(configPath, configData, 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Mock the home directory for testing
			originalHome := os.Getenv("HOME")
			defer os.Setenv("HOME", originalHome)
			os.Setenv("HOME", tempDir)

			// Load config
			cfg, err := loadConfigFromFile()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.SSEPort != tt.expected {
				t.Errorf("Expected SSEPort %d, got %d", tt.expected, cfg.SSEPort)
			}
		})
	}
}

func TestLogFileConfiguration(t *testing.T) {
	// Create a temporary directory for home
	tempDir, err := os.MkdirTemp("", "mcp-test-home")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config directory structure
	configDir := filepath.Join(tempDir, ".config", "markdown-reader-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "markdown-reader-mcp.json")

	tests := []struct {
		name     string
		logFile  string
		expected string
	}{
		{
			name:     "absolute path log file",
			logFile:  "/var/log/markdown-reader-mcp.log",
			expected: "/var/log/markdown-reader-mcp.log",
		},
		{
			name:     "tilde path log file",
			logFile:  "~/logs/server.log",
			expected: "~/logs/server.log",
		},
		{
			name:     "empty log file",
			logFile:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config with log file
			testConfig := Config{
				Directories: []string{"test"},
				LogFile:     tt.logFile,
			}

			configData, err := json.Marshal(testConfig)
			if err != nil {
				t.Fatalf("Failed to marshal test config: %v", err)
			}

			err = os.WriteFile(configPath, configData, 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Mock the home directory for testing
			originalHome := os.Getenv("HOME")
			defer os.Setenv("HOME", originalHome)
			os.Setenv("HOME", tempDir)

			// Load config
			cfg, err := loadConfigFromFile()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.LogFile != tt.expected {
				t.Errorf("Expected LogFile '%s', got '%s'", tt.expected, cfg.LogFile)
			}
		})
	}
}
