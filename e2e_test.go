package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Test configuration
const (
	testTimeout = 10 * time.Second
)

// MCPTestClient represents a test client for the MCP server
type MCPTestClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	reader *bufio.Reader
}

// NewMCPTestClient creates a new test client
func NewMCPTestClient(t *testing.T) *MCPTestClient {
	// Build the server if needed
	buildCmd := exec.Command("go", "build", "-o", "markdown-reader-mcp", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}

	// Start the server with test directories
	cmd := exec.Command("./markdown-reader-mcp", "./test_data", ".")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	client := &MCPTestClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		reader: bufio.NewReader(stdout),
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	return client
}

// Close closes the test client
func (c *MCPTestClient) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	c.stderr.Close()

	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}

	return c.cmd.Wait()
}

// SendRequest sends a JSON-RPC request and returns the response
func (c *MCPTestClient) SendRequest(request any) (map[string]any, error) {
	// Serialize request
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	if _, err := c.stdin.Write(append(requestBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response with timeout
	responseChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	go func() {
		line, _, err := c.reader.ReadLine()
		if err != nil {
			errorChan <- err
			return
		}
		responseChan <- string(line)
	}()

	select {
	case response := <-responseChan:
		// Parse JSON response
		var result map[string]any
		if err := json.Unmarshal([]byte(response), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		return result, nil
	case err := <-errorChan:
		return nil, fmt.Errorf("failed to read response: %w", err)
	case <-time.After(testTimeout):
		return nil, fmt.Errorf("request timed out")
	}
}

// Test helper functions
func createInitializeRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"resources": map[string]any{},
				"tools":     map[string]any{},
			},
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
}

func createResourceListRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "resources/list",
		"params":  map[string]any{},
	}
}

func createResourceReadRequest(id int, uri string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "resources/read",
		"params": map[string]any{
			"uri": uri,
		},
	}
}

func createToolListRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/list",
		"params":  map[string]any{},
	}
}

func createToolCallRequest(id int, name string, arguments map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	}
}

// Test functions

func TestServerInitialization(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	// Test initialization
	response, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

	// Verify response structure
	if response["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc 2.0, got %v", response["jsonrpc"])
	}

	if response["id"].(float64) != 1 {
		t.Errorf("Expected id 1, got %v", response["id"])
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object, got %T", response["result"])
	}

	// Check server info
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("Expected serverInfo object")
	}

	if serverInfo["name"] != "Markdown Reader" {
		t.Errorf("Expected server name 'Markdown Reader', got %v", serverInfo["name"])
	}

	// Check capabilities
	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("Expected capabilities object")
	}

	if _, hasResources := capabilities["resources"]; !hasResources {
		t.Error("Expected resources capability")
	}

	if _, hasTools := capabilities["tools"]; !hasTools {
		t.Error("Expected tools capability")
	}
}

func TestResourcesList(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	// Initialize first
	_, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// List resources
	response, err := client.SendRequest(createResourceListRequest(2))
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	resources, ok := result["resources"].([]any)
	if !ok {
		t.Fatalf("Expected resources array")
	}

	// Verify we have the expected resources
	expectedResources := map[string]bool{
		"markdown://find_all": false,
	}

	for _, resource := range resources {
		res := resource.(map[string]any)
		uri := res["uri"].(string)
		if _, exists := expectedResources[uri]; exists {
			expectedResources[uri] = true
		}
	}

	for uri, found := range expectedResources {
		if !found {
			t.Errorf("Expected resource %s not found", uri)
		}
	}
}

func TestMarkdownFilesList(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	// Initialize
	_, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Read markdown list resource
	response, err := client.SendRequest(createResourceReadRequest(2, "markdown://find_all"))
	if err != nil {
		t.Fatalf("Failed to read markdown list: %v", err)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	contents, ok := result["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("Expected contents array")
	}

	content := contents[0].(map[string]any)
	text := content["text"].(string)

	// Parse the JSON response
	var listData map[string]any
	if err := json.Unmarshal([]byte(text), &listData); err != nil {
		t.Fatalf("Failed to parse markdown list JSON: %v", err)
	}

	// Verify structure
	files, ok := listData["files"].([]any)
	if !ok {
		t.Fatalf("Expected files array")
	}

	// Should have at least the test files we created
	if len(files) < 5 {
		t.Errorf("Expected at least 5 markdown files, got %d", len(files))
	}

	// Check that we have files from test_data
	foundTestFile := false
	for _, file := range files {
		fileData := file.(map[string]any)
		path := fileData["path"].(string)
		if strings.Contains(path, "test_data") {
			foundTestFile = true
			break
		}
	}

	if !foundTestFile {
		t.Error("Expected to find files from test_data directory")
	}
}

func TestMarkdownFileRead(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	// Initialize
	_, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test reading a specific markdown file using the tool
	testFile := "test_data/docs/api.md"
	response, err := client.SendRequest(createToolCallRequest(2, "read_markdown_file", map[string]any{
		"file_path": testFile,
	}))
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("Expected content array")
	}

	textContent := content[0].(map[string]any)
	text := textContent["text"].(string)

	// Verify content contains expected text
	if !strings.Contains(text, "# API Documentation") {
		t.Error("Expected file content to contain API Documentation header")
	}
}

func TestMarkdownFileReadByName(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	// Initialize
	_, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test reading a file by just its name (should find api.md in test_data/docs/)
	response, err := client.SendRequest(createToolCallRequest(2, "read_markdown_file", map[string]any{
		"file_path": "api.md",
	}))
	if err != nil {
		t.Fatalf("Failed to read markdown file by name: %v", err)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("Expected content array")
	}

	textContent := content[0].(map[string]any)
	text := textContent["text"].(string)

	// Verify content contains expected text
	if !strings.Contains(text, "# API Documentation") {
		t.Error("Expected file content to contain API Documentation header")
	}

	// Test reading a file by name without extension
	response, err = client.SendRequest(createToolCallRequest(3, "read_markdown_file", map[string]any{
		"file_path": "README",
	}))
	if err != nil {
		t.Fatalf("Failed to read README by name: %v", err)
	}

	result, ok = response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	content, ok = result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("Expected content array")
	}

	textContent = content[0].(map[string]any)
	text = textContent["text"].(string)

	// Verify content contains expected text from test_data/README.md
	if !strings.Contains(text, "# Test Data") {
		t.Error("Expected file content to contain Test Data header")
	}
}

func TestToolsList(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	// Initialize
	_, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// List tools
	response, err := client.SendRequest(createToolListRequest(2))
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("Expected tools array")
	}

	// Verify expected tools
	expectedTools := map[string]bool{
		"read_markdown_file": false,
	}

	for _, tool := range tools {
		toolData := tool.(map[string]any)
		name := toolData["name"].(string)
		if _, exists := expectedTools[name]; exists {
			expectedTools[name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool %s not found", name)
		}
	}
}

func TestErrorHandling(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	// Initialize
	_, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Test reading non-existent file using the tool
	response, err := client.SendRequest(createToolCallRequest(2, "read_markdown_file", map[string]any{
		"file_path": "nonexistent.md",
	}))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Should get a result with error content
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("Expected content array")
	}

	textContent := content[0].(map[string]any)
	text := textContent["text"].(string)

	// Should contain error message about file not found
	if !strings.Contains(strings.ToLower(text), "file not found") && !strings.Contains(strings.ToLower(text), "failed to read file") {
		t.Errorf("Expected error message about file not found or failed to read file, got: %s", text)
	}
}

// Test runner
func TestMain(m *testing.M) {
	// Setup
	log.SetOutput(io.Discard) // Suppress logs during tests

	// Run tests
	code := m.Run()

	os.Exit(code)
}
