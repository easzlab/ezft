package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config *DownloadConfig
		want   bool
	}{
		{
			name:   "with nil config",
			config: nil,
			want:   true,
		},
		{
			name: "with custom config",
			config: &DownloadConfig{
				URL:            "http://example.com/file.zip",
				OutputPath:     "test.zip",
				ChunkSize:      2048,
				MaxConcurrency: 2,
				RetryCount:     5,
				EnableResume:   false,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			if (client != nil) != tt.want {
				t.Errorf("NewClient() = %v, want %v", client != nil, tt.want)
			}

			if client != nil {
				if client.config == nil {
					t.Error("Client config should not be nil")
				}
				if client.httpClient == nil {
					t.Error("HTTP client should not be nil")
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if config.ChunkSize != 1024*1024 {
		t.Errorf("Expected ChunkSize to be %d, got %d", 1024*1024, config.ChunkSize)
	}

	if config.MaxConcurrency != 1 {
		t.Errorf("Expected MaxConcurrency to be 1, got %d", config.MaxConcurrency)
	}

	if config.RetryCount != 3 {
		t.Errorf("Expected RetryCount to be 3, got %d", config.RetryCount)
	}

	if !config.EnableResume {
		t.Error("Expected EnableResume to be true")
	}
}

func TestGetFileInfo(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Content-Length", "1024")
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
		case "GET":
			if r.Header.Get("Range") == "bytes=0-0" {
				w.WriteHeader(http.StatusPartialContent)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL: server.URL + "/test.txt",
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	size, supportsRange, err := client.getFileInfo(ctx)

	if err != nil {
		t.Fatalf("getFileInfo() error = %v", err)
	}

	if size != 1024 {
		t.Errorf("Expected file size 1024, got %d", size)
	}

	if !supportsRange {
		t.Error("Expected server to support range requests")
	}
}

func TestGetFileInfoError(t *testing.T) {
	// Test with invalid URL
	config := &DownloadConfig{
		URL: "http://invalid-url-that-does-not-exist.com/file.txt",
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	_, _, err := client.getFileInfo(ctx)

	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestGetExistingFileSize(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	// Test non-existing file
	config := &DownloadConfig{
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	size, err := client.getExistingFileSize()
	if err != nil {
		t.Fatalf("getExistingFileSize() error = %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0 for non-existing file, got %d", size)
	}

	// Create test file
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	size, err = client.getExistingFileSize()
	if err != nil {
		t.Fatalf("getExistingFileSize() error = %v", err)
	}
	expectedSize := int64(len(testContent))
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestGetProgress(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "progress_test.txt")

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
		FileSize:   1000,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	// Test with no existing file
	progress, err := client.GetProgress()
	if err != nil {
		t.Fatalf("GetProgress() error = %v", err)
	}
	if progress != 0 {
		t.Errorf("Expected progress 0%%, got %.2f%%", progress)
	}

	// Create partial file
	partialContent := make([]byte, 500)
	err = os.WriteFile(testFile, partialContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	// Test with partial file
	progress, err = client.GetProgress()
	if err != nil {
		t.Fatalf("GetProgress() error = %v", err)
	}
	expectedProgress := 50.0
	if progress != expectedProgress {
		t.Errorf("Expected progress %.2f%%, got %.2f%%", expectedProgress, progress)
	}
}

func TestDownloadBasic(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "download_test.txt")
	testContent := "This is test content for download"

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Content-Length", "33")
			w.WriteHeader(http.StatusOK)
		case "GET":
			w.Header().Set("Content-Length", "33")
			w.Write([]byte(testContent))
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:          server.URL + "/test.txt",
		OutputPath:   testFile,
		EnableResume: false,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.Download(ctx)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Downloaded content mismatch. Expected %q, got %q", testContent, string(content))
	}
}

func TestDownloadWithContext(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "context_test.txt")

	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Length", "10")
		w.Write([]byte("test data"))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:          server.URL + "/test.txt",
		OutputPath:   testFile,
		EnableResume: false,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.Download(ctx)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestDownloadAlreadyComplete(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "complete_test.txt")
	testContent := "Already downloaded content"

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
			w.WriteHeader(http.StatusOK)
		case "GET":
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testContent))
		}
	}))
	defer server.Close()

	// Create file with exact size
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err = client.Download(ctx)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	// Verify file wasn't changed
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content changed unexpectedly. Expected %q, got %q", testContent, string(content))
	}
}
