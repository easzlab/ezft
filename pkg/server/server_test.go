package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name string
		root string
		port int
		want *Server
	}{
		{
			name: "valid_server",
			root: "/tmp/test",
			port: 8080,
			want: &Server{root: "/tmp/test", port: 8080},
		},
		{
			name: "empty_root",
			root: "",
			port: 9000,
			want: &Server{root: "", port: 9000},
		},
		{
			name: "zero_port",
			root: "/var/www",
			port: 0,
			want: &Server{root: "/var/www", port: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewServer(tt.root, tt.port)
			if got.root != tt.want.root {
				t.Errorf("NewServer() root = %v, want %v", got.root, tt.want.root)
			}
			if got.port != tt.want.port {
				t.Errorf("NewServer() port = %v, want %v", got.port, tt.want.port)
			}
		})
	}
}

func TestServer_Start(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Find available port
	port := findAvailablePort(t)

	server := NewServer(tempDir, port)
	server.SetLogger(zap.NewNop())

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test server is running
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	// Test file serving
	resp, err := http.Get(baseURL + "/test.txt")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != testContent {
		t.Errorf("Expected body %q, got %q", testContent, string(body))
	}

	// Test directory listing
	resp, err = http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("Failed to make request to root: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for directory listing, got %d", resp.StatusCode)
	}

	// Check if server stops (with timeout)
	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		// Server should still be running, which is expected for this test
		t.Log("Server is running as expected")
	}
}

func TestServer_StartInvalidPort(t *testing.T) {
	tempDir := t.TempDir()

	// Test with invalid port (negative)
	server := NewServer(tempDir, -1)
	server.SetLogger(zap.NewNop())

	err := server.Start()
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

func TestServer_StartPortInUse(t *testing.T) {
	tempDir := t.TempDir()
	port := findAvailablePort(t)

	// Start first server
	server1 := NewServer(tempDir, port)
	server1.SetLogger(zap.NewNop())
	go func() {
		server1.Start()
	}()

	// Wait for first server to start
	time.Sleep(100 * time.Millisecond)

	// Try to start second server on same port
	server2 := NewServer(tempDir, port)
	server2.SetLogger(zap.NewNop())
	err := server2.Start()

	if err == nil {
		t.Error("Expected error when port is in use, got nil")
	}
}

func TestServer_StartNonExistentDirectory(t *testing.T) {
	// Use non-existent directory
	nonExistentDir := "/non/existent/directory"
	port := findAvailablePort(t)

	server := NewServer(nonExistentDir, port)
	server.SetLogger(zap.NewNop())

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test request to non-existent directory
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 404 or similar error for non-existent directory
	if resp.StatusCode == http.StatusOK {
		t.Error("Expected non-200 status for non-existent directory")
	}
}

func TestServer_StartWithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create subdirectory with file
	subDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subDir, "subfile.txt")
	subContent := "Subdirectory content"
	err = os.WriteFile(subFile, []byte(subContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create subfile: %v", err)
	}

	port := findAvailablePort(t)
	server := NewServer(tempDir, port)
	server.SetLogger(zap.NewNop())

	// Start server in goroutine
	go func() {
		server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test subdirectory file access
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	resp, err := http.Get(baseURL + "/subdir/subfile.txt")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != subContent {
		t.Errorf("Expected body %q, got %q", subContent, string(body))
	}
}

func TestServer_StartWithMiddleware(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "middleware_test.txt")
	testContent := "Middleware test content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	port := findAvailablePort(t)
	server := NewServer(tempDir, port)
	server.SetLogger(zap.NewNop())

	// Start server in goroutine
	go func() {
		server.Start()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test that middleware is applied (LoggingMiddleware should be active)
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	resp, err := http.Get(baseURL + "/middleware_test.txt")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify content is served correctly through middleware
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != testContent {
		t.Errorf("Expected body %q, got %q", testContent, string(body))
	}
}

// Helper function to find an available port
func findAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}
