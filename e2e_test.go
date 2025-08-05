package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

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

func NewMCPTestClient(t *testing.T) *MCPTestClient {
	// Start the server with test directories
	cmd := exec.Command("./markdown-reader-mcp", "./test/dir1", ".")

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

	return client
}

func (c *MCPTestClient) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	c.stderr.Close()

	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}

	return c.cmd.Wait()
}

func (c *MCPTestClient) SendRequest(request any) (map[string]any, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := c.stdin.Write(append(requestBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

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

func createResourceReadRequest(id int, uri string) map[string]any {
	params := map[string]any{
		"uri": uri,
	}
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "resources/read",
		"params":  params,
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

func TestServerInitialization(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	response, err := client.SendRequest(createInitializeRequest(1))
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}

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

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("Expected serverInfo object")
	}

	if serverInfo["name"] != "Markdown Reader" {
		t.Errorf("Expected server name 'Markdown Reader', got %v", serverInfo["name"])
	}

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

func TestE2EFindMarkdownFiles(t *testing.T) {
	client := NewMCPTestClient(t)
	defer client.Close()

	initializeMCPServer(t, client)
	toolResponse := findAllMarkdownFilesToolCall(t, client)
	filesList := parseToolResponseToFilesList(t, toolResponse)
	filenames := convertFilesToFilenames(t, filesList)

	assertMinimumFileCount(t, filesList, 5)
	assertRequiredFilesArePresent(t, filenames, []string{"foo.md", "bar.md", "baz.md"})
}

func parseToolResponseToFilesList(t *testing.T, response map[string]any) []any {
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object in response")
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("Expected non-empty content array")
	}

	textContent := content[0].(map[string]any)
	jsonText := textContent["text"].(string)

	var responseData map[string]any
	if err := json.Unmarshal([]byte(jsonText), &responseData); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	files, ok := responseData["files"].([]any)
	if !ok {
		t.Fatalf("Expected files array in response data")
	}

	return files
}

func convertFilesToFilenames(t *testing.T, files []any) []string {
	var filenames []string
	for _, file := range files {
		fileObj, ok := file.(map[string]any)
		if !ok {
			t.Fatalf("Expected file object, got %T", file)
		}

		name, ok := fileObj["name"].(string)
		if !ok {
			t.Fatalf("Expected filename string, got %T", fileObj["name"])
		}

		filenames = append(filenames, name)
	}
	return filenames
}

func assertMinimumFileCount(t *testing.T, files []any, minCount int) {
	if len(files) < minCount {
		t.Errorf("Expected at least %d markdown files, got %d", minCount, len(files))
	}
}

func initializeMCPServer(t *testing.T, client *MCPTestClient) {
	if _, err := client.SendRequest(createInitializeRequest(1)); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
}

func findAllMarkdownFilesToolCall(t *testing.T, client *MCPTestClient) map[string]any {
	response, err := client.SendRequest(createToolCallRequest(2, "find_markdown_files", map[string]any{}))
	if err != nil {
		t.Fatalf("Failed to call find_markdown_files tool: %v", err)
	}
	return response
}

func assertRequiredFilesArePresent(t *testing.T, actualFilenames []string, requiredFiles []string) {
	for _, requiredFile := range requiredFiles {
		if !slices.Contains(actualFilenames, requiredFile) {
			t.Errorf("Expected to find %s in results", requiredFile)
		}
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

	// Test reading a specific markdown file using the resource
	response, err := client.SendRequest(createResourceReadRequest(2, "file://bar.md"))
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("Expected result object")
	}

	contents, ok := result["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("Expected contents array")
	}

	resourceContent := contents[0].(map[string]any)
	text := resourceContent["text"].(string)

	// Verify content contains expected text
	if !strings.Contains(text, "# Bar") {
		t.Error("Expected file content to contain bar header")
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
		"find_markdown_files": false,
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

	// Test reading non-existent file using the resource
	response, err := client.SendRequest(createResourceReadRequest(2, "file://nonexistent.md"))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Should get an error in the JSON-RPC response
	if errorObj, hasError := response["error"]; hasError {
		errorMap := errorObj.(map[string]any)
		message := errorMap["message"].(string)
		if !strings.Contains(strings.ToLower(message), "file not found") {
			t.Errorf("Expected error message about file not found, got: %s", message)
		}
	} else {
		t.Fatal("Expected error response for non-existent file but got success")
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
