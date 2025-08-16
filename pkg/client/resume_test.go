package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestDownloadWithResume(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "resume_test.txt")
	failedChunksFile := testFile + ".failed_chunks.json"
	fullContent := "This is the complete file content for resume testing"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			// Full file request
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fullContent))
			return
		}

		// Handle range requests
		w.WriteHeader(http.StatusPartialContent)
		switch rangeHeader {
		case "bytes=0-9":
			w.Write([]byte(fullContent[0:10]))
		case "bytes=10-19":
			w.Write([]byte(fullContent[10:20]))
		case "bytes=20-29":
			w.Write([]byte(fullContent[20:30]))
		case "bytes=30-39":
			w.Write([]byte(fullContent[30:40]))
		case "bytes=40-49":
			w.Write([]byte(fullContent[40:50]))
		case "bytes=50-51":
			// Fix: return the correct remaining bytes (2 bytes: "ng")
			w.Write([]byte(fullContent[50:]))
		default:
			// For any other range, return empty to avoid "default" content
			w.Write([]byte{})
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:               server.URL + "/test.txt",
		OutputPath:        testFile,
		FailedChunksJason: failedChunksFile,
		ChunkSize:         10,
		MaxConcurrency:    1,
		EnableResume:      true,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop()) // Add logger initialization

	ctx := context.Background()
	err := client.downloadWithResume(ctx, int64(len(fullContent)))
	if err != nil {
		t.Fatalf("downloadWithResume() error = %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != fullContent {
		t.Errorf("Downloaded content mismatch. Expected %q, got %q", fullContent, string(content))
	}

	// Verify no failed chunks file exists
	if _, err := os.Stat(failedChunksFile); err == nil {
		t.Error("Failed chunks file should not exist after successful download")
	}
}

func TestDownloadWithResumePartialFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "partial_resume_test.txt")
	fullContent := "This is the complete file content for partial resume testing"
	partialContent := fullContent[:20] // First 20 bytes already downloaded

	// Create partial file
	err := os.WriteFile(testFile, []byte(partialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		w.WriteHeader(http.StatusPartialContent)

		// Only serve remaining content
		switch rangeHeader {
		case "bytes=20-29":
			w.Write([]byte(fullContent[20:30]))
		case "bytes=30-39":
			w.Write([]byte(fullContent[30:40]))
		case "bytes=40-49":
			w.Write([]byte(fullContent[40:50]))
		case "bytes=50-59":
			w.Write([]byte(fullContent[50:]))
		default:
			w.Write([]byte("default"))
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:            server.URL + "/test.txt",
		OutputPath:     testFile,
		ChunkSize:      10,
		MaxConcurrency: 1,
		EnableResume:   true,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop())

	ctx := context.Background()
	err = client.downloadWithResume(ctx, int64(len(fullContent)))
	if err != nil {
		t.Fatalf("downloadWithResume() error = %v", err)
	}

	// Verify complete file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != fullContent {
		t.Errorf("Downloaded content mismatch. Expected %q, got %q", fullContent, string(content))
	}
}

func TestDownloadWithResumeFailedChunks(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "failed_chunks_resume_test.txt")
	failedChunksFile := testFile + ".failed_chunks.json"

	// Create failed chunks file
	failedChunks := []Chunk{
		{Index: 1, Start: 10, End: 19},
		{Index: 3, Start: 30, End: 39},
	}
	err := (&Client{config: &DownloadConfig{FailedChunksJason: failedChunksFile}}).saveFailedChunks(failedChunks)
	if err != nil {
		t.Fatalf("Failed to create failed chunks file: %v", err)
	}

	fullContent := "This is the complete file content for failed chunks testing"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		w.WriteHeader(http.StatusPartialContent)

		switch rangeHeader {
		case "bytes=10-19":
			w.Write([]byte(fullContent[10:20]))
		case "bytes=30-39":
			w.Write([]byte(fullContent[30:40]))
		default:
			w.Write([]byte("default"))
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:               server.URL + "/test.txt",
		OutputPath:        testFile,
		FailedChunksJason: failedChunksFile,
		ChunkSize:         10,
		MaxConcurrency:    1,
		EnableResume:      true,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop())

	// Create partial file (missing the failed chunks)
	partialContent := make([]byte, len(fullContent))
	copy(partialContent, fullContent)
	// Zero out the failed chunk areas
	for i := 10; i < 20; i++ {
		partialContent[i] = 0
	}
	for i := 30; i < 40; i++ {
		partialContent[i] = 0
	}
	err = os.WriteFile(testFile, partialContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	ctx := context.Background()
	err = client.downloadWithResume(ctx, int64(len(fullContent)))
	if err != nil {
		t.Fatalf("downloadWithResume() error = %v", err)
	}

	// Verify failed chunks file was deleted
	if _, err := os.Stat(failedChunksFile); err == nil {
		t.Error("Failed chunks file should be deleted after successful download")
	}
}

func TestDownloadChunksSequentially(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "sequential_test.txt")
	testContent := "Sequential download test content"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		w.WriteHeader(http.StatusPartialContent)

		switch rangeHeader {
		case "bytes=0-9":
			w.Write([]byte(testContent[0:10]))
		case "bytes=10-19":
			w.Write([]byte(testContent[10:20]))
		case "bytes=20-31":
			w.Write([]byte(testContent[20:]))
		default:
			w.Write([]byte("default"))
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL: server.URL + "/test.txt",
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop())

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	chunks := []Chunk{
		{Index: 0, Start: 0, End: 9},
		{Index: 1, Start: 10, End: 19},
		{Index: 2, Start: 20, End: 31},
	}

	ctx := context.Background()
	err = client.downloadChunksSequentially(ctx, file, chunks)
	if err != nil {
		t.Fatalf("downloadChunksSequentially() error = %v", err)
	}

	// Verify file content
	file.Seek(0, 0)
	content := make([]byte, len(testContent))
	n, err := file.Read(content)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content[:n]) != testContent {
		t.Errorf("Downloaded content mismatch. Expected %q, got %q", testContent, string(content[:n]))
	}
}

func TestDownloadChunksSequentiallyWithFailure(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "sequential_fail_test.txt")
	failedChunksFile := testFile + ".failed_chunks.json"

	t.Logf("Test file: %s", testFile)
	t.Logf("Failed chunks file: %s", failedChunksFile)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")

		// Fail the second chunk
		if rangeHeader == "bytes=10-19" {
			t.Logf("Simulating failure for range: %s", rangeHeader)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:               server.URL + "/test.txt",
		FailedChunksJason: failedChunksFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop())

	// Verify config is set correctly
	t.Logf("Client config FailedChunksJason: %s", client.config.FailedChunksJason)

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	chunks := []Chunk{
		{Index: 0, Start: 0, End: 9},
		{Index: 1, Start: 10, End: 19}, // This will fail
		{Index: 2, Start: 20, End: 29},
	}

	ctx := context.Background()
	err = client.downloadChunksSequentially(ctx, file, chunks)

	// Should return error due to failed chunk
	if err == nil {
		t.Error("Expected error due to failed chunk")
	} else {
		t.Logf("Got expected error: %v", err)
	}

	// Check if failed chunks file exists
	if _, err := os.Stat(failedChunksFile); os.IsNotExist(err) {
		t.Error("Failed chunks file should exist after failed download")

		// List files in temp directory for debugging
		files, _ := os.ReadDir(tempDir)
		t.Logf("Files in temp directory:")
		for _, file := range files {
			t.Logf("  %s", file.Name())
		}
	} else {
		t.Logf("Failed chunks file exists: %s", failedChunksFile)

		// Read and log the content
		content, readErr := os.ReadFile(failedChunksFile)
		if readErr == nil {
			t.Logf("Failed chunks file content: %s", string(content))
		}

		// Clean up failed chunks file for test isolation
		if err := os.Remove(failedChunksFile); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: failed to clean up failed chunks file: %v", err)
		}
	}
}

func TestDownloadWithResumeCompleteFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "complete_file_test.txt")
	testContent := "Complete file content"

	// Create complete file
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create complete file: %v", err)
	}

	config := &DownloadConfig{
		URL:        "http://example.com/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop())

	ctx := context.Background()
	err = client.downloadWithResume(ctx, int64(len(testContent)))
	if err != nil {
		t.Fatalf("downloadWithResume() error = %v", err)
	}

	// Verify file content unchanged
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content should be unchanged. Expected %q, got %q", testContent, string(content))
	}
}

func TestDownloadWithResumeCreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "nested", "dir", "resume_test.txt")
	testContent := "Test content for nested directory in resume"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		w.WriteHeader(http.StatusPartialContent)

		switch rangeHeader {
		case "bytes=0-9":
			w.Write([]byte(testContent[0:10]))
		case "bytes=10-19":
			w.Write([]byte(testContent[10:20]))
		case "bytes=20-29":
			w.Write([]byte(testContent[20:30]))
		case "bytes=30-39":
			w.Write([]byte(testContent[30:40]))
		case "bytes=40-42":
			w.Write([]byte(testContent[40:]))
		default:
			// For any other range, return appropriate slice
			if len(testContent) > 10 {
				w.Write([]byte(testContent[0:10]))
			} else {
				w.Write([]byte(testContent))
			}
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:            server.URL + "/test.txt",
		OutputPath:     nestedPath,
		ChunkSize:      10,
		MaxConcurrency: 1,
	}
	client := NewClient(config)
	client.SetLogger(zap.NewNop())

	ctx := context.Background()
	err := client.downloadWithResume(ctx, int64(len(testContent)))
	if err != nil {
		t.Fatalf("downloadWithResume() error = %v", err)
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
