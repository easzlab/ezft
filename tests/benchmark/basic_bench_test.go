package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/easzlab/ezft/pkg/client"
)

// BenchmarkBasicDownload benchmarks basic download functionality
func BenchmarkBasicDownload(b *testing.B) {
	// Create test content of different sizes
	testSizes := []struct {
		name string
		size int
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, tc := range testSizes {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkBasicDownloadSize(b, tc.size)
		})
	}
}

func benchmarkBasicDownloadSize(b *testing.B, size int) {
	// Create test content
	testContent := make([]byte, size)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	b.ResetTimer()
	b.SetBytes(int64(size))

	for i := 0; i < b.N; i++ {
		tempDir := b.TempDir()
		testFile := filepath.Join(tempDir, fmt.Sprintf("bench_test_%d.bin", i))

		config := &client.DownloadConfig{
			URL:        server.URL + "/test.bin",
			OutputPath: testFile,
		}
		cli := client.NewClient(config)

		ctx := context.Background()
		err := cli.BasicDownload(ctx)
		if err != nil {
			b.Fatalf("BasicDownload() error = %v", err)
		}

		// Verify file size
		info, err := os.Stat(testFile)
		if err != nil {
			b.Fatalf("Failed to stat file: %v", err)
		}
		if info.Size() != int64(size) {
			b.Fatalf("File size mismatch: expected %d, got %d", size, info.Size())
		}
	}
}

// BenchmarkCopyWithOptimizedBuffer benchmarks the optimized copy function
func BenchmarkCopyWithOptimizedBuffer(b *testing.B) {
	testSizes := []struct {
		name string
		size int
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	chunkSizes := []int64{
		64 * 1024,       // 64KB
		1024 * 1024,     // 1MB
		2 * 1024 * 1024, // 2MB
	}

	for _, tc := range testSizes {
		for _, chunkSize := range chunkSizes {
			b.Run(fmt.Sprintf("%s_Chunk%dKB", tc.name, chunkSize/1024), func(b *testing.B) {
				benchmarkCopyWithOptimizedBuffer(b, tc.size, chunkSize)
			})
		}
	}
}

func benchmarkCopyWithOptimizedBuffer(b *testing.B, size int, chunkSize int64) {
	// Create test content
	testContent := make([]byte, size)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	config := &client.DownloadConfig{
		URL:       server.URL + "/test.bin",
		ChunkSize: chunkSize,
	}
	cli := client.NewClient(config)

	b.ResetTimer()
	b.SetBytes(int64(size))

	for i := 0; i < b.N; i++ {
		tempDir := b.TempDir()
		testFile := filepath.Join(tempDir, fmt.Sprintf("bench_copy_%d.bin", i))

		file, err := os.Create(testFile)
		if err != nil {
			b.Fatalf("Failed to create file: %v", err)
		}

		resp, err := http.Get(server.URL + "/test.bin")
		if err != nil {
			file.Close()
			b.Fatalf("Failed to get response: %v", err)
		}

		ctx := context.Background()
		_, err = cli.CopyWithOptimizedBuffer(ctx, file, resp.Body)
		resp.Body.Close()
		file.Close()

		if err != nil {
			b.Fatalf("CopyWithOptimizedBuffer() error = %v", err)
		}
	}
}
