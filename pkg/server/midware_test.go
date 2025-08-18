package server

import (
	"bytes"
	"encoding/base64"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"status_200", http.StatusOK},
		{"status_404", http.StatusNotFound},
		{"status_500", http.StatusInternalServerError},
		{"status_201", http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: recorder,
				statusCode:     200, // default
			}

			rw.WriteHeader(tt.statusCode)

			if rw.statusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, rw.statusCode)
			}

			if recorder.Code != tt.statusCode {
				t.Errorf("Expected recorder status code %d, got %d", tt.statusCode, recorder.Code)
			}
		})
	}
}

func TestResponseWriter_Write(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantLen int
	}{
		{"empty_data", []byte{}, 0},
		{"small_data", []byte("hello"), 5},
		{"large_data", bytes.Repeat([]byte("a"), 1024), 1024},
		{"unicode_data", []byte("你好世界"), 12}, // UTF-8 encoding
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: recorder,
				statusCode:     200,
				responseSize:   0,
			}

			n, err := rw.Write(tt.data)
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			if n != tt.wantLen {
				t.Errorf("Write() returned %d, want %d", n, tt.wantLen)
			}

			if rw.responseSize != int64(tt.wantLen) {
				t.Errorf("responseSize = %d, want %d", rw.responseSize, tt.wantLen)
			}

			if !bytes.Equal(recorder.Body.Bytes(), tt.data) {
				t.Errorf("Body = %v, want %v", recorder.Body.Bytes(), tt.data)
			}
		})
	}
}

func TestResponseWriter_MultipleWrites(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     200,
		responseSize:   0,
	}

	// Multiple writes should accumulate response size
	data1 := []byte("Hello, ")
	data2 := []byte("World!")

	n1, err := rw.Write(data1)
	if err != nil {
		t.Fatalf("First Write() error = %v", err)
	}

	n2, err := rw.Write(data2)
	if err != nil {
		t.Fatalf("Second Write() error = %v", err)
	}

	expectedSize := int64(n1 + n2)
	if rw.responseSize != expectedSize {
		t.Errorf("responseSize = %d, want %d", rw.responseSize, expectedSize)
	}

	expectedBody := "Hello, World!"
	if recorder.Body.String() != expectedBody {
		t.Errorf("Body = %q, want %q", recorder.Body.String(), expectedBody)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr) // Restore original output

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	// Wrap with logging middleware
	handler := server.LoggingMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "http://example.com")
	req.RemoteAddr = "192.168.1.1:12345"

	recorder := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(recorder, req)

	// Verify response
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "test response" {
		t.Errorf("Expected body 'test response', got %q", recorder.Body.String())
	}

	// Note: Since we're using zap.NewNop(), there won't be any log output to verify
	// This test mainly ensures the middleware doesn't crash and passes requests through correctly
}

func TestLoggingMiddleware_WithError(t *testing.T) {
	// Create test handler that returns error
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	handler := server.LoggingMiddleware(testHandler)

	req := httptest.NewRequest("POST", "/error", strings.NewReader("request body"))
	req.Header.Set("Content-Length", "12")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Verify error response
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", recorder.Code)
	}
}

func TestLoggingMiddleware_EmptyHeaders(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	handler := server.LoggingMiddleware(testHandler)

	// Request without User-Agent and Referer headers
	req := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Should handle empty headers gracefully
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

func TestAuthMiddleware_ValidCredentials(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	handler := server.AuthMiddleware(testHandler)

	// Create request with valid credentials
	req := httptest.NewRequest("GET", "/protected", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("admin:password"))
	req.Header.Set("Authorization", "Basic "+auth)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Should allow access with valid credentials
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "authenticated" {
		t.Errorf("Expected body 'authenticated', got %q", recorder.Body.String())
	}
}

func TestAuthMiddleware_InvalidCredentials(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	handler := server.AuthMiddleware(testHandler)

	// Create request with invalid credentials
	req := httptest.NewRequest("GET", "/protected", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword"))
	req.Header.Set("Authorization", "Basic "+auth)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Should deny access with invalid credentials
	if recorder.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", recorder.Code)
	}
}

func TestAuthMiddleware_NoCredentials(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	handler := server.AuthMiddleware(testHandler)

	// Create request without credentials
	req := httptest.NewRequest("GET", "/protected", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Should require authentication
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}

	// Should set WWW-Authenticate header
	authHeader := recorder.Header().Get("WWW-Authenticate")
	if authHeader == "" {
		t.Error("Expected WWW-Authenticate header to be set")
	}
}

func TestAuthMiddleware_MalformedAuth(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	handler := server.AuthMiddleware(testHandler)

	// Create request with malformed authorization header
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic invalidbase64")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Should require proper authentication
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}
}

func TestMiddlewareChaining(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	// Chain both middlewares: Logging -> Auth -> Handler
	handler := server.LoggingMiddleware(server.AuthMiddleware(testHandler))

	// Create request with valid credentials
	req := httptest.NewRequest("GET", "/protected", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("admin:password"))
	req.Header.Set("Authorization", "Basic "+auth)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Should work with both middlewares
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %q", recorder.Body.String())
	}
}

func TestMiddlewareChaining_AuthFailure(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Create server instance and set logger
	server := NewServer("/tmp", 8080)
	server.SetLogger(zap.NewNop())

	// Chain both middlewares: Logging -> Auth -> Handler
	handler := server.LoggingMiddleware(server.AuthMiddleware(testHandler))

	// Create request without credentials
	req := httptest.NewRequest("GET", "/protected", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Should fail at auth middleware
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}
}
