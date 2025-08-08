package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewServer(t *testing.T) {
	server := NewServer("/tmp", 8080)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.root != "/tmp" {
		t.Errorf("Expected root '/tmp', got '%s'", server.root)
	}
	if server.port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.port)
	}
}

func TestHandleHealth(t *testing.T) {
	server := NewServer("/tmp", 8080)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "status") || !strings.Contains(body, "ok") {
		t.Errorf("Expected health response to contain status and ok, got: %s", body)
	}
}

func TestHandleDownload(t *testing.T) {
	// 创建临时目录和测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World! This is a test file for download."

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := NewServer(tempDir, 8080)

	t.Run("download complete file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download/test.txt", nil)
		w := httptest.NewRecorder()

		server.handleDownload(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if body != testContent {
			t.Errorf("Expected content '%s', got '%s'", testContent, body)
		}

		// 检查响应头
		if w.Header().Get("Accept-Ranges") != "bytes" {
			t.Error("Expected Accept-Ranges header to be 'bytes'")
		}

		if w.Header().Get("Content-Type") != "application/octet-stream" {
			t.Error("Expected Content-Type to be 'application/octet-stream'")
		}
	})

	t.Run("download with range request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download/test.txt", nil)
		req.Header.Set("Range", "bytes=0-4")
		w := httptest.NewRecorder()

		server.handleDownload(w, req)

		if w.Code != http.StatusPartialContent {
			t.Errorf("Expected status 206, got %d", w.Code)
		}

		body := w.Body.String()
		expected := testContent[:5] // "Hello"
		if body != expected {
			t.Errorf("Expected content '%s', got '%s'", expected, body)
		}

		contentRange := w.Header().Get("Content-Range")
		expectedRange := fmt.Sprintf("bytes 0-4/%d", len(testContent))
		if contentRange != expectedRange {
			t.Errorf("Expected Content-Range '%s', got '%s'", expectedRange, contentRange)
		}
	})

	t.Run("download non-existent file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download/nonexistent.txt", nil)
		w := httptest.NewRecorder()

		server.handleDownload(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("download with invalid path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download/../../../etc/passwd", nil)
		w := httptest.NewRecorder()

		server.handleDownload(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("download directory", func(t *testing.T) {
		// 创建子目录
		subDir := filepath.Join(tempDir, "subdir")
		err := os.Mkdir(subDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		req := httptest.NewRequest("GET", "/download/subdir", nil)
		w := httptest.NewRecorder()

		server.handleDownload(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleFileInfo(t *testing.T) {
	// 创建临时目录和测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "info_test.txt")
	testContent := "File info test content"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := NewServer(tempDir, 8080)

	t.Run("get file info", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/info/info_test.txt", nil)
		w := httptest.NewRecorder()

		server.handleFileInfo(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		body := w.Body.String()
		if !strings.Contains(body, "name") || !strings.Contains(body, "size") {
			t.Errorf("Expected file info to contain name and size, got: %s", body)
		}
	})

	t.Run("get info for non-existent file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/info/nonexistent.txt", nil)
		w := httptest.NewRecorder()

		server.handleFileInfo(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		name        string
		rangeHeader string
		size        int64
		expected    []Range
		expectError bool
	}{
		{
			name:        "simple range",
			rangeHeader: "bytes=0-499",
			size:        1000,
			expected:    []Range{{start: 0, end: 499}},
			expectError: false,
		},
		{
			name:        "suffix range",
			rangeHeader: "bytes=-500",
			size:        1000,
			expected:    []Range{{start: 500, end: 999}},
			expectError: false,
		},
		{
			name:        "prefix range",
			rangeHeader: "bytes=500-",
			size:        1000,
			expected:    []Range{{start: 500, end: 999}},
			expectError: false,
		},
		{
			name:        "invalid range header",
			rangeHeader: "invalid",
			size:        1000,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid range values",
			rangeHeader: "bytes=500-400",
			size:        1000,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRange(tt.rangeHeader, tt.size)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d ranges, got %d", len(tt.expected), len(result))
				return
			}

			for i, r := range result {
				if r.start != tt.expected[i].start || r.end != tt.expected[i].end {
					t.Errorf("Expected range %d: {%d, %d}, got {%d, %d}",
						i, tt.expected[i].start, tt.expected[i].end, r.start, r.end)
				}
			}
		})
	}
}

func TestServerIntegration(t *testing.T) {
	// 创建临时目录和大文件
	tempDir := t.TempDir()
	largeFile := filepath.Join(tempDir, "large.txt")

	// 创建1MB的测试文件
	content := bytes.Repeat([]byte("A"), 1024*1024)
	err := os.WriteFile(largeFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	server := NewServer(tempDir, 8080)

	// 测试服务器创建
	if server == nil {
		t.Fatal("Failed to create server")
	}

	// 测试大文件的部分下载
	t.Run("large file partial download", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/download/large.txt", nil)
		req.Header.Set("Range", "bytes=0-1023") // 下载前1KB
		w := httptest.NewRecorder()

		server.handleDownload(w, req)

		if w.Code != http.StatusPartialContent {
			t.Errorf("Expected status 206, got %d", w.Code)
		}

		if w.Body.Len() != 1024 {
			t.Errorf("Expected 1024 bytes, got %d", w.Body.Len())
		}

		// 验证内容
		expectedContent := bytes.Repeat([]byte("A"), 1024)
		if !bytes.Equal(w.Body.Bytes(), expectedContent) {
			t.Error("Downloaded content doesn't match expected")
		}
	})
}

func BenchmarkHandleDownload(b *testing.B) {
	// 创建临时目录和测试文件
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "bench.txt")
	testContent := bytes.Repeat([]byte("benchmark test content "), 1000)

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	server := NewServer(tempDir, 8080)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/download/bench.txt", nil)
		w := httptest.NewRecorder()

		server.handleDownload(w, req)

		if w.Code != http.StatusOK {
			b.Errorf("Expected status 200, got %d", w.Code)
		}
	}
}
