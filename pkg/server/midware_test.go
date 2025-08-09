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

	// Wrap with logging middleware
	handler := LoggingMiddleware(testHandler)

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

	// Verify log output
	logOutput := logBuffer.String()
	if logOutput == "" {
		t.Error("Expected log output, got empty string")
	}

	// Check log contains expected information
	expectedParts := []string{
		"192.168.1.1:12345",
		"GET",
		"/test?param=value",
		"Status: 200",
		"RespSize: 13 bytes", // "test response" is 13 bytes
		"UserAgent: \"test-agent\"",
		"Referer: \"http://example.com\"",
	}

	for _, part := range expectedParts {
		if !strings.Contains(logOutput, part) {
			t.Errorf("Log output missing expected part: %q\nActual log: %s", part, logOutput)
		}
	}
}

func TestLoggingMiddleware_WithError(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr)

	// Create test handler that returns error
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	handler := LoggingMiddleware(testHandler)

	req := httptest.NewRequest("POST", "/error", strings.NewReader("request body"))
	req.Header.Set("Content-Length", "12")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Verify error response
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", recorder.Code)
	}

	// Verify log contains error status
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Status: 500") {
		t.Errorf("Log should contain 'Status: 500', got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "ReqSize: 12 bytes") {
		t.Errorf("Log should contain request size, got: %s", logOutput)
	}
}

func TestLoggingMiddleware_EmptyHeaders(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(testHandler)

	// Request without User-Agent and Referer headers
	req := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	logOutput := logBuffer.String()
	
	// Should handle empty headers gracefully
	if !strings.Contains(logOutput, "UserAgent: \"\"") {
		t.Errorf("Log should contain empty UserAgent, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "Referer: \"\"") {
		t.Errorf("Log should contain empty Referer, got: %s", logOutput)
	}
}

func TestAuthMiddleware_ValidCredentials(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	handler := AuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	
	// Set valid Basic Auth credentials (admin:password)
	auth := base64.StdEncoding.EncodeToString([]byte("admin:password"))
	req.Header.Set("Authorization", "Basic "+auth)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "authenticated" {
		t.Errorf("Expected body 'authenticated', got %q", recorder.Body.String())
	}
}

func TestAuthMiddleware_InvalidCredentials(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid credentials")
	})

	handler := AuthMiddleware(testHandler)

	tests := []struct {
		name     string
		username string
		password string
		wantCode int
	}{
		{"wrong_username", "wronguser", "password", http.StatusForbidden},
		{"wrong_password", "admin", "wrongpass", http.StatusForbidden},
		{"both_wrong", "wronguser", "wrongpass", http.StatusForbidden},
		{"empty_username", "", "password", http.StatusForbidden},
		{"empty_password", "admin", "", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			
			auth := base64.StdEncoding.EncodeToString([]byte(tt.username + ":" + tt.password))
			req.Header.Set("Authorization", "Basic "+auth)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, recorder.Code)
			}
		})
	}
}

func TestAuthMiddleware_NoCredentials(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without credentials")
	})

	handler := AuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/protected", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}

	// Check WWW-Authenticate header
	authHeader := recorder.Header().Get("WWW-Authenticate")
	expectedAuth := `Basic realm="Restricted"`
	if authHeader != expectedAuth {
		t.Errorf("Expected WWW-Authenticate header %q, got %q", expectedAuth, authHeader)
	}
}

func TestAuthMiddleware_InvalidAuthFormat(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid auth format")
	})

	handler := AuthMiddleware(testHandler)

	tests := []struct {
		name   string
		header string
	}{
		{"no_basic_prefix", "invalid-auth-header"},
		{"malformed_basic", "Basic invalid-base64"},
		{"empty_basic", "Basic "},
		{"bearer_token", "Bearer token123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tt.header)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", recorder.Code)
			}
		})
	}
}

func TestMiddlewareChain(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Chain both middlewares: Auth -> Logging -> Handler
	handler := LoggingMiddleware(AuthMiddleware(testHandler))

	// Test with valid credentials
	req := httptest.NewRequest("GET", "/protected", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("admin:password"))
	req.Header.Set("Authorization", "Basic "+auth)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %q", recorder.Body.String())
	}

	// Verify logging middleware captured the request
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Status: 200") {
		t.Errorf("Log should contain successful status, got: %s", logOutput)
	}
}

func TestMiddlewareChain_AuthFailure(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	defer log.SetOutput(os.Stderr)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when auth fails")
	})

	// Chain both middlewares
	handler := LoggingMiddleware(AuthMiddleware(testHandler))

	// Test without credentials
	req := httptest.NewRequest("GET", "/protected", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}

	// Verify logging middleware captured the auth failure
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "Status: 401") {
		t.Errorf("Log should contain auth failure status, got: %s", logOutput)
	}
}