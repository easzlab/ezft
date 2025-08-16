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
	"go.uber.org/zap"
)

// BenchmarkConcurrentDownload benchmarks concurrent download functionality
func BenchmarkConcurrentDownload(b *testing.B) {
	// Test different file sizes
	testSizes := []struct {
		name string
		size int
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	// Test different concurrency levels
	concurrencyLevels := []int{1, 2, 4, 8, 16}

	for _, tc := range testSizes {
		for _, concurrency := range concurrencyLevels {
			b.Run(fmt.Sprintf("%s_Concurrency%d", tc.name, concurrency), func(b *testing.B) {
				benchmarkConcurrentDownloadSize(b, tc.size, concurrency)
			})
		}
	}
}

func benchmarkConcurrentDownloadSize(b *testing.B, size, concurrency int) {
	// Create test content
	testContent := make([]byte, size)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	// Create test server with Range support
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.Header().Set("Accept-Ranges", "bytes")

		// Handle Range requests
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			// Simple range parsing for benchmark (assumes bytes=start-end format)
			var start, end int
			if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); n == 2 {
				if start >= 0 && end < len(testContent) && start <= end {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(testContent)))
					w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
					w.WriteHeader(http.StatusPartialContent)
					w.Write(testContent[start : end+1])
					return
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	b.ResetTimer()
	b.SetBytes(int64(size))

	for i := 0; i < b.N; i++ {
		tempDir := b.TempDir()
		testFile := filepath.Join(tempDir, fmt.Sprintf("bench_concurrent_%d.bin", i))

		config := &client.DownloadConfig{
			URL:            server.URL + "/test.bin",
			OutputPath:     testFile,
			MaxConcurrency: concurrency,
			ChunkSize:      1024 * 1024, // 1MB chunks
			EnableResume:   true,
			AutoChunk:      false,
		}
		client := client.NewClient(config)
		client.SetLogger(zap.NewNop())

		ctx := context.Background()
		err := client.Download(ctx)
		if err != nil {
			b.Fatalf("Download() error = %v", err)
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

// BenchmarkChunkSizeImpact benchmarks the impact of different chunk sizes
func BenchmarkChunkSizeImpact(b *testing.B) {
	fileSize := 50 * 1024 * 1024 // 50MB
	testContent := make([]byte, fileSize)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	// Test different chunk sizes
	chunkSizes := []struct {
		name string
		size int64
	}{
		{"64KB", 64 * 1024},
		{"256KB", 256 * 1024},
		{"1MB", 1024 * 1024},
		{"4MB", 4 * 1024 * 1024},
		{"16MB", 16 * 1024 * 1024},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.Header().Set("Accept-Ranges", "bytes")

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			var start, end int
			if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); n == 2 {
				if start >= 0 && end < len(testContent) && start <= end {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(testContent)))
					w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
					w.WriteHeader(http.StatusPartialContent)
					w.Write(testContent[start : end+1])
					return
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	for _, tc := range chunkSizes {
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(fileSize))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tempDir := b.TempDir()
				testFile := filepath.Join(tempDir, fmt.Sprintf("bench_chunk_%d.bin", i))

				config := &client.DownloadConfig{
					URL:            server.URL + "/test.bin",
					OutputPath:     testFile,
					MaxConcurrency: 4,
					ChunkSize:      tc.size,
					EnableResume:   true,
					AutoChunk:      false,
				}
				client := client.NewClient(config)
				client.SetLogger(zap.NewNop())

				ctx := context.Background()
				err := client.Download(ctx)
				if err != nil {
					b.Fatalf("Download() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkConcurrentVsBasicDownload compares concurrent vs basic download performance
func BenchmarkConcurrentVsBasicDownload(b *testing.B) {
	fileSize := 20 * 1024 * 1024 // 20MB
	testContent := make([]byte, fileSize)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.Header().Set("Accept-Ranges", "bytes")

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			var start, end int
			if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); n == 2 {
				if start >= 0 && end < len(testContent) && start <= end {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(testContent)))
					w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
					w.WriteHeader(http.StatusPartialContent)
					w.Write(testContent[start : end+1])
					return
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	b.Run("BasicDownload", func(b *testing.B) {
		b.SetBytes(int64(fileSize))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			tempDir := b.TempDir()
			testFile := filepath.Join(tempDir, fmt.Sprintf("bench_basic_%d.bin", i))

			config := &client.DownloadConfig{
				URL:            server.URL + "/test.bin",
				OutputPath:     testFile,
				MaxConcurrency: 1,
				EnableResume:   false,
			}
			client := client.NewClient(config)
			client.SetLogger(zap.NewNop())

			ctx := context.Background()
			err := client.BasicDownload(ctx)
			if err != nil {
				b.Fatalf("BasicDownload() error = %v", err)
			}
		}
	})

	b.Run("ConcurrentDownload", func(b *testing.B) {
		b.SetBytes(int64(fileSize))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			tempDir := b.TempDir()
			testFile := filepath.Join(tempDir, fmt.Sprintf("bench_concurrent_%d.bin", i))

			config := &client.DownloadConfig{
				URL:            server.URL + "/test.bin",
				OutputPath:     testFile,
				MaxConcurrency: 8,
				ChunkSize:      2 * 1024 * 1024, // 2MB chunks
				EnableResume:   true,
				AutoChunk:      false,
			}
			client := client.NewClient(config)
			client.SetLogger(zap.NewNop())

			ctx := context.Background()
			err := client.Download(ctx)
			if err != nil {
				b.Fatalf("Download() error = %v", err)
			}
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	fileSize := 100 * 1024 * 1024 // 100MB
	testContent := make([]byte, fileSize)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.Header().Set("Accept-Ranges", "bytes")

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			var start, end int
			if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); n == 2 {
				if start >= 0 && end < len(testContent) && start <= end {
					w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(testContent)))
					w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
					w.WriteHeader(http.StatusPartialContent)
					w.Write(testContent[start : end+1])
					return
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	memoryProfiles := []struct {
		name        string
		concurrency int
		chunkSize   int64
	}{
		{"LowMemory", 2, 256 * 1024},       // 2 goroutines, 256KB chunks
		{"MediumMemory", 4, 1024 * 1024},   // 4 goroutines, 1MB chunks
		{"HighMemory", 8, 4 * 1024 * 1024}, // 8 goroutines, 4MB chunks
	}

	for _, profile := range memoryProfiles {
		b.Run(profile.name, func(b *testing.B) {
			b.SetBytes(int64(fileSize))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tempDir := b.TempDir()
				testFile := filepath.Join(tempDir, fmt.Sprintf("bench_memory_%d.bin", i))

				config := &client.DownloadConfig{
					URL:            server.URL + "/test.bin",
					OutputPath:     testFile,
					MaxConcurrency: profile.concurrency,
					ChunkSize:      profile.chunkSize,
					EnableResume:   true,
					AutoChunk:      false,
				}
				client := client.NewClient(config)
				client.SetLogger(zap.NewNop())

				ctx := context.Background()
				err := client.Download(ctx)
				if err != nil {
					b.Fatalf("Download() error = %v", err)
				}
			}
		})
	}
}
