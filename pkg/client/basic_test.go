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

func TestBasicDownload(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "basic_test.txt")
	testContent := "This is a test file content for basic download"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		// Use io.WriteString to ensure complete write
		_, err := w.Write([]byte(testContent))
		if err != nil {
			t.Logf("Error writing response: %v", err)
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.BasicDownload(ctx)
	if err != nil {
		t.Fatalf("BasicDownload() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Downloaded file does not exist")
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

func TestBasicDownloadServerError(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "error_test.txt")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.BasicDownload(ctx)
	if err == nil {
		t.Error("Expected error for server error response")
	}

	// Verify file was not created or is empty
	if info, err := os.Stat(testFile); err == nil && info.Size() > 0 {
		t.Error("File should not exist or should be empty after failed download")
	}
}

func TestBasicDownloadNotFound(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "notfound_test.txt")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/nonexistent.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.BasicDownload(ctx)
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestBasicDownloadContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "cancel_test.txt")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This should not be downloaded"))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.BasicDownload(ctx)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestBasicDownloadCreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "nested", "dir", "test.txt")
	testContent := "Test content for nested directory"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: nestedPath,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.BasicDownload(ctx)
	if err != nil {
		t.Fatalf("BasicDownload() error = %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(nestedPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Nested directory should have been created")
	}

	// Verify file content
	content, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Downloaded content mismatch. Expected %q, got %q", testContent, string(content))
	}
}

func TestBasicDownloadOverwriteExisting(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "overwrite_test.txt")
	originalContent := "Original content"
	newContent := "New downloaded content"

	// Create existing file
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(newContent))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err = client.BasicDownload(ctx)
	if err != nil {
		t.Fatalf("BasicDownload() error = %v", err)
	}

	// Verify file was overwritten
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != newContent {
		t.Errorf("File should be overwritten. Expected %q, got %q", newContent, string(content))
	}
}

func TestBasicDownloadLargeFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "large_test.txt")

	// Create large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.WriteHeader(http.StatusOK)
		w.Write(largeContent)
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/large.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	startTime := time.Now()
	err := client.BasicDownload(ctx)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("BasicDownload() error = %v", err)
	}

	// Verify file size
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat downloaded file: %v", err)
	}

	if info.Size() != int64(len(largeContent)) {
		t.Errorf("Downloaded file size mismatch. Expected %d, got %d", len(largeContent), info.Size())
	}

	t.Logf("Downloaded 1MB file in %v", duration)
}

func TestBasicDownloadInvalidURL(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid_test.txt")

	config := &DownloadConfig{
		URL:        "http://invalid-domain-that-does-not-exist.com/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.BasicDownload(ctx)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestBasicDownloadUserAgent(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "useragent_test.txt")

	var receivedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.BasicDownload(ctx)
	if err != nil {
		t.Fatalf("BasicDownload() error = %v", err)
	}

	expectedUserAgent := "Mozilla/5.0 (compatible; ezft/1.0)"
	if receivedUserAgent != expectedUserAgent {
		t.Errorf("Expected User-Agent %q, got %q", expectedUserAgent, receivedUserAgent)
	}
}

// TestGetOptimalBufferSize tests the buffer size calculation function
func TestGetOptimalBufferSize(t *testing.T) {
	tests := []struct {
		name           string
		chunkSize      int64
		expectedResult int64
	}{
		{
			name:           "Small chunk size - should use minimum",
			chunkSize:      1024,
			expectedResult: 64 * 1024, // 64KB minimum
		},
		{
			name:           "Normal chunk size",
			chunkSize:      512 * 1024, // 512KB
			expectedResult: 512 * 1024,
		},
		{
			name:           "Large chunk size - should use maximum",
			chunkSize:      10 * 1024 * 1024, // 10MB
			expectedResult: 2 * 1024 * 1024,  // 2MB maximum
		},
		{
			name:           "Default chunk size",
			chunkSize:      1024 * 1024, // 1MB
			expectedResult: 1024 * 1024,
		},
		{
			name:           "Zero chunk size - should use minimum",
			chunkSize:      0,
			expectedResult: 64 * 1024, // 64KB minimum
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DownloadConfig{
				ChunkSize: tt.chunkSize,
			}
			client := NewClient(config)
			client.SetLogger(zap.NewNop()) // Add logger initialization

			result := client.getOptimalBufferSize()
			if result != tt.expectedResult {
				t.Errorf("getOptimalBufferSize() = %d, expected %d", result, tt.expectedResult)
			}
		})
	}
}

// TestBasicDownloadRetry tests the retry mechanism
func TestBasicDownloadRetry(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "retry_test.txt")
	testContent := "Success after retry"

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first 2 attempts
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
		RetryCount: 3, // Allow 3 retries
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	startTime := time.Now()
	err := client.BasicDownload(ctx)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("BasicDownload() should succeed after retries, error = %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Downloaded content mismatch. Expected %q, got %q", testContent, string(content))
	}

	// Verify retry attempts
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Verify exponential backoff (should take at least 1+2=3 seconds for backoff)
	if duration < 3*time.Second {
		t.Errorf("Expected duration >= 3s due to exponential backoff, got %v", duration)
	}

	t.Logf("Download succeeded after %d attempts in %v", attemptCount, duration)
}

// TestBasicDownloadRetryExhausted tests when all retry attempts are exhausted
func TestBasicDownloadRetryExhausted(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "retry_exhausted_test.txt")

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		// Always fail
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
		RetryCount: 2, // Allow 2 retries
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.BasicDownload(ctx)

	// Should fail after all retries are exhausted
	if err == nil {
		t.Error("Expected error after all retries exhausted")
	}

	// Verify total attempts (initial + retries)
	expectedAttempts := 3 // 1 initial + 2 retries
	if attemptCount != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attemptCount)
	}

	// Verify error message contains retry information
	if err != nil && err.Error() == "" {
		t.Error("Error message should not be empty")
	}

	t.Logf("Failed as expected after %d attempts with error: %v", attemptCount, err)
}

// TestBasicDownloadRetryContextCancellation tests retry with context cancellation
func TestBasicDownloadRetryContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "retry_cancel_test.txt")

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		// Always fail to trigger retry
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		OutputPath: testFile,
		RetryCount: 5, // Allow many retries
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	// Create context with short timeout to cancel during retry
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	err := client.BasicDownload(ctx)

	// Should fail due to context cancellation
	if err == nil {
		t.Error("Expected context cancellation error")
	}

	// Should have made at least one attempt but not all retries
	if attemptCount == 0 {
		t.Error("Should have made at least one attempt")
	}
	if attemptCount > 3 {
		t.Errorf("Should not have made too many attempts due to context cancellation, got %d", attemptCount)
	}

	t.Logf("Cancelled after %d attempts with error: %v", attemptCount, err)
}
