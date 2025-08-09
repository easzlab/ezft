package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDownloadChunksConcurrently(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "concurrent_test.txt")
	failedChunksFile := testFile + ".failed_chunks.json"

	// Track requests to verify concurrency
	var requestMutex sync.Mutex
	requestTimes := make(map[string]time.Time)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMutex.Lock()
		requestTimes[r.Header.Get("Range")] = time.Now()
		requestMutex.Unlock()

		// Simulate some processing time
		time.Sleep(50 * time.Millisecond)

		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			http.Error(w, "Range header required", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusPartialContent)
		
		// Return different content based on range
		switch rangeHeader {
		case "bytes=0-9":
			w.Write([]byte("0123456789"))
		case "bytes=10-19":
			w.Write([]byte("abcdefghij"))
		case "bytes=20-29":
			w.Write([]byte("ABCDEFGHIJ"))
		default:
			w.Write([]byte("1234567890"))
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:               server.URL + "/test.txt",
		OutputPath:        testFile,
		FailedChunksJason: failedChunksFile,
		MaxConcurrency:    3,
		RetryCount:        1,
	}
	client := NewClient(config)

	// Create test file
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	// Create test chunks
	chunks := []Chunk{
		{Index: 0, Start: 0, End: 9},
		{Index: 1, Start: 10, End: 19},
		{Index: 2, Start: 20, End: 29},
	}

	ctx := context.Background()
	startTime := time.Now()
	err = client.downloadChunksConcurrently(ctx, file, chunks)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("downloadChunksConcurrently() error = %v", err)
	}

	// Verify concurrency - should take less time than sequential
	maxSequentialTime := time.Duration(len(chunks)) * 50 * time.Millisecond
	if duration >= maxSequentialTime {
		t.Errorf("Download took too long (%v), expected concurrent execution", duration)
	}

	// Verify file content
	file.Seek(0, 0)
	content := make([]byte, 30)
	n, err := file.Read(content)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := "0123456789abcdefghijABCDEFGHIJ"
	if string(content[:n]) != expected {
		t.Errorf("Expected %q, got %q", expected, string(content[:n]))
	}

	// Verify no failed chunks file exists
	if _, err := os.Stat(failedChunksFile); err == nil {
		t.Error("Failed chunks file should not exist after successful download")
	}
}

func TestDownloadChunksConcurrentlyWithFailures(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "concurrent_fail_test.txt")
	failedChunksFile := testFile + ".failed_chunks.json"

	failureCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		
		// Fail specific chunks
		if rangeHeader == "bytes=10-19" {
			failureCount++
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusPartialContent)
		switch rangeHeader {
		case "bytes=0-9":
			w.Write([]byte("0123456789"))
		case "bytes=20-29":
			w.Write([]byte("ABCDEFGHIJ"))
		default:
			w.Write([]byte("1234567890"))
		}
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:               server.URL + "/test.txt",
		OutputPath:        testFile,
		FailedChunksJason: failedChunksFile,
		MaxConcurrency:    2,
		RetryCount:        1,
	}
	client := NewClient(config)

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
	err = client.downloadChunksConcurrently(ctx, file, chunks)
	
	// Should return error due to failed chunk
	if err == nil {
		t.Error("Expected error due to failed chunk")
	}

	// Verify failed chunks file was created
	if _, err := os.Stat(failedChunksFile); os.IsNotExist(err) {
		t.Error("Failed chunks file should exist after failed download")
	}

	// Verify failed chunks content
	failedChunks, err := client.loadFailedChunks()
	if err != nil {
		t.Fatalf("Failed to load failed chunks: %v", err)
	}

	if len(failedChunks) != 1 {
		t.Errorf("Expected 1 failed chunk, got %d", len(failedChunks))
	}

	if len(failedChunks) > 0 && failedChunks[0].Index != 1 {
		t.Errorf("Expected failed chunk index 1, got %d", failedChunks[0].Index)
	}
}

func TestDownloadChunksConcurrentlyContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "context_cancel_test.txt")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:            server.URL + "/test.txt",
		OutputPath:     testFile,
		MaxConcurrency: 2,
	}
	client := NewClient(config)

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	chunks := []Chunk{
		{Index: 0, Start: 0, End: 9},
		{Index: 1, Start: 10, End: 19},
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = client.downloadChunksConcurrently(ctx, file, chunks)
	
	// Should return context error
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestDownloadChunksConcurrentlyMaxConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "max_concurrency_test.txt")

	var activeConcurrency int32
	var maxConcurrency int32
	var mutex sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		activeConcurrency++
		if activeConcurrency > maxConcurrency {
			maxConcurrency = activeConcurrency
		}
		mutex.Unlock()

		// Simulate processing time
		time.Sleep(100 * time.Millisecond)

		mutex.Lock()
		activeConcurrency--
		mutex.Unlock()

		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:            server.URL + "/test.txt",
		OutputPath:     testFile,
		MaxConcurrency: 2, // Limit to 2 concurrent requests
	}
	client := NewClient(config)

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	// Create 5 chunks to test concurrency limit
	chunks := make([]Chunk, 5)
	for i := 0; i < 5; i++ {
		chunks[i] = Chunk{
			Index: int64(i),
			Start: int64(i * 10),
			End:   int64(i*10 + 9),
		}
	}

	ctx := context.Background()
	err = client.downloadChunksConcurrently(ctx, file, chunks)
	if err != nil {
		t.Fatalf("downloadChunksConcurrently() error = %v", err)
	}

	// Verify max concurrency was respected
	if maxConcurrency > 2 {
		t.Errorf("Max concurrency exceeded: expected <= 2, got %d", maxConcurrency)
	}

	if maxConcurrency < 2 {
		t.Errorf("Concurrency not utilized: expected 2, got %d", maxConcurrency)
	}
}

func TestDownloadChunksConcurrentlyEmptyChunks(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "empty_chunks_test.txt")

	config := &DownloadConfig{
		URL:        "http://example.com/test.txt",
		OutputPath: testFile,
	}
	client := NewClient(config)

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	// Test with empty chunks slice
	chunks := []Chunk{}

	ctx := context.Background()
	err = client.downloadChunksConcurrently(ctx, file, chunks)
	if err != nil {
		t.Fatalf("downloadChunksConcurrently() with empty chunks error = %v", err)
	}
}