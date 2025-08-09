package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateChunks(t *testing.T) {
	config := &DownloadConfig{
		ChunkSize: 1024,
		AutoChunk: false,
	}
	client := NewClient(config)

	tests := []struct {
		name      string
		start     int64
		end       int64
		chunkSize int64
		expected  int
	}{
		{
			name:      "exact division",
			start:     0,
			end:       2048,
			chunkSize: 1024,
			expected:  2,
		},
		{
			name:      "with remainder",
			start:     0,
			end:       2500,
			chunkSize: 1024,
			expected:  3,
		},
		{
			name:      "single chunk",
			start:     0,
			end:       500,
			chunkSize: 1024,
			expected:  1,
		},
		{
			name:      "partial range",
			start:     1000,
			end:       3000,
			chunkSize: 1024,
			expected:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.config.ChunkSize = tt.chunkSize
			chunks := client.calculateChunks(tt.start, tt.end)

			if len(chunks) != tt.expected {
				t.Errorf("Expected %d chunks, got %d", tt.expected, len(chunks))
			}

			// Verify chunk boundaries
			for i, chunk := range chunks {
				if i == 0 && chunk.Start != tt.start {
					t.Errorf("First chunk start should be %d, got %d", tt.start, chunk.Start)
				}
				if i == len(chunks)-1 && chunk.End != tt.end-1 {
					t.Errorf("Last chunk end should be %d, got %d", tt.end-1, chunk.End)
				}
				if chunk.Index != int64(i) {
					t.Errorf("Chunk %d index should be %d, got %d", i, i, chunk.Index)
				}
			}
		})
	}
}

func TestCalculateChunksAutoChunk(t *testing.T) {
	config := &DownloadConfig{
		AutoChunk: true,
	}
	client := NewClient(config)

	tests := []struct {
		name     string
		fileSize int64
		expected int64 // expected chunk size
	}{
		{
			name:     "small file",
			fileSize: 50 * 1024 * 1024, // 50MB
			expected: 4 * 1024 * 1024,  // 4MB
		},
		{
			name:     "medium file",
			fileSize: 500 * 1024 * 1024, // 500MB
			expected: 10 * 1024 * 1024,  // 10MB
		},
		{
			name:     "large file",
			fileSize: 5 * 1024 * 1024 * 1024, // 5GB
			expected: 20 * 1024 * 1024,       // 20MB
		},
		{
			name:     "very large file",
			fileSize: 50 * 1024 * 1024 * 1024, // 50GB
			expected: 50 * 1024 * 1024,        // 50MB
		},
		{
			name:     "huge file",
			fileSize: 500 * 1024 * 1024 * 1024, // 500GB
			expected: 100 * 1024 * 1024,        // 100MB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := client.calculateChunks(0, tt.fileSize)

			if client.config.ChunkSize != tt.expected {
				t.Errorf("Expected chunk size %d, got %d", tt.expected, client.config.ChunkSize)
			}

			// Verify total coverage
			if len(chunks) > 0 {
				lastChunk := chunks[len(chunks)-1]
				if lastChunk.End != tt.fileSize-1 {
					t.Errorf("Last chunk should end at %d, got %d", tt.fileSize-1, lastChunk.End)
				}
			}
		})
	}
}

func TestCalculateChunkSize(t *testing.T) {
	tests := []struct {
		name      string
		totalSize int64
		expected  int64
	}{
		{
			name:      "small file",
			totalSize: 50 * 1024 * 1024, // 50MB
			expected:  4 * 1024 * 1024,  // 4MB
		},
		{
			name:      "medium file",
			totalSize: 500 * 1024 * 1024, // 500MB
			expected:  10 * 1024 * 1024,  // 10MB
		},
		{
			name:      "large file",
			totalSize: 5 * 1024 * 1024 * 1024, // 5GB
			expected:  20 * 1024 * 1024,       // 20MB
		},
		{
			name:      "very large file",
			totalSize: 50 * 1024 * 1024 * 1024, // 50GB
			expected:  50 * 1024 * 1024,        // 50MB
		},
		{
			name:      "huge file",
			totalSize: 500 * 1024 * 1024 * 1024, // 500GB
			expected:  100 * 1024 * 1024,        // 100MB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateChunkSize(tt.totalSize)
			if result != tt.expected {
				t.Errorf("calculateChunkSize(%d) = %d, expected %d", tt.totalSize, result, tt.expected)
			}
		})
	}
}

func TestSaveAndLoadFailedChunks(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	failedChunksFile := testFile + ".failed_chunks.json"

	config := &DownloadConfig{
		OutputPath:        testFile,
		FailedChunksJason: failedChunksFile,
	}
	client := NewClient(config)

	// Test chunks to save
	originalChunks := []Chunk{
		{Index: 0, Start: 0, End: 1023},
		{Index: 2, Start: 2048, End: 3071},
		{Index: 5, Start: 5120, End: 6143},
	}

	// Test saving failed chunks
	err := client.saveFailedChunks(originalChunks)
	if err != nil {
		t.Fatalf("saveFailedChunks() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(failedChunksFile); os.IsNotExist(err) {
		t.Fatal("Failed chunks file was not created")
	}

	// Test loading failed chunks
	loadedChunks, err := client.loadFailedChunks()
	if err != nil {
		t.Fatalf("loadFailedChunks() error = %v", err)
	}

	// Verify loaded chunks match original
	if len(loadedChunks) != len(originalChunks) {
		t.Fatalf("Expected %d chunks, got %d", len(originalChunks), len(loadedChunks))
	}

	for i, chunk := range loadedChunks {
		expected := originalChunks[i]
		if chunk.Index != expected.Index || chunk.Start != expected.Start || chunk.End != expected.End {
			t.Errorf("Chunk %d mismatch. Expected %+v, got %+v", i, expected, chunk)
		}
	}
}

func TestLoadFailedChunksNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "nonexistent.txt")
	failedChunksFile := testFile + ".failed_chunks.json"

	config := &DownloadConfig{
		OutputPath:        testFile,
		FailedChunksJason: failedChunksFile,
	}
	client := NewClient(config)

	// Test loading from non-existent file
	chunks, err := client.loadFailedChunks()
	if err != nil {
		t.Fatalf("loadFailedChunks() error = %v", err)
	}

	if len(chunks) != 0 {
		t.Errorf("Expected empty chunks for non-existent file, got %d chunks", len(chunks))
	}
}

func TestLoadFailedChunksInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	failedChunksFile := testFile + ".failed_chunks.json"

	// Create invalid JSON file
	err := os.WriteFile(failedChunksFile, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	config := &DownloadConfig{
		OutputPath:        testFile,
		FailedChunksJason: failedChunksFile,
	}
	client := NewClient(config)

	// Test loading invalid JSON
	_, err = client.loadFailedChunks()
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestDownloadChunkOnce(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "chunk_test.txt")
	testContent := "This is test content for chunk download testing"

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			http.Error(w, "Range header required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Range", "bytes 0-9/"+string(rune(len(testContent))))
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte(testContent[:10])) // Return first 10 bytes
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL: server.URL + "/test.txt",
	}
	client := NewClient(config)

	// Create test file
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	// Test downloading a chunk
	chunk := Chunk{
		Index: 0,
		Start: 0,
		End:   9,
	}

	ctx := context.Background()
	err = client.downloadChunkOnce(ctx, file, chunk)
	if err != nil {
		t.Fatalf("downloadChunkOnce() error = %v", err)
	}

	// Verify file content
	file.Seek(0, 0)
	buffer := make([]byte, 10)
	n, err := file.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expected := testContent[:10]
	if string(buffer[:n]) != expected {
		t.Errorf("Expected %q, got %q", expected, string(buffer[:n]))
	}
}

func TestDownloadChunkWithRetry(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "retry_test.txt")

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail first 2 attempts
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		RetryCount: 3,
	}
	client := NewClient(config)

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	chunk := Chunk{
		Index: 0,
		Start: 0,
		End:   6,
	}

	ctx := context.Background()
	err = client.downloadChunk(ctx, file, chunk)
	if err != nil {
		t.Fatalf("downloadChunk() error = %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestDownloadChunkMaxRetries(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "max_retry_test.txt")

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Always fail
		http.Error(w, "Server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &DownloadConfig{
		URL:        server.URL + "/test.txt",
		RetryCount: 2,
	}
	client := NewClient(config)

	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	chunk := Chunk{
		Index: 0,
		Start: 0,
		End:   6,
	}

	ctx := context.Background()
	err = client.downloadChunk(ctx, file, chunk)
	if err == nil {
		t.Error("Expected error after max retries")
	}

	expectedAttempts := 3 // initial attempt + 2 retries
	if attempts != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attempts)
	}
}
