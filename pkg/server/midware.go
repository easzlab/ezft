package server

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter wraps the original ResponseWriter to capture response information
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(size)
	return size, err
}

// Logging middleware
func (s *Server) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create custom ResponseWriter to capture response information
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     200, // default status code
			responseSize:   0,
		}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Get request information
		duration := time.Since(start)
		userAgent := r.Header.Get("User-Agent")
		referer := r.Header.Get("Referer")
		contentLength := r.ContentLength
		if contentLength < 0 {
			contentLength = 0
		}

		// Log detailed information
		s.logger.Info("",
			zap.String("time", start.Format("2006-01-02 15:04:05")),
			zap.String("remoteAddr", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.String("url", r.URL.RequestURI()), // Use RequestURI to include query parameters
			zap.Int("statusCode", rw.statusCode),
			zap.Int64("reqSize", contentLength),
			zap.Int64("respSize", rw.responseSize),
			zap.Duration("duration", duration),
			zap.String("userAgent", userAgent),
			zap.String("referer", referer),
		)
	})
}

// Authentication middleware
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Basic Auth credentials from request headers
		username, password, ok := r.BasicAuth()
		if !ok {
			// If no authentication information is provided, require authentication
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			s.logger.Warn("Unauthorized request",
				zap.String("remoteAddr", r.RemoteAddr),
				zap.String("url", r.URL.RequestURI()))
			return
		}

		// Check username and password (hardcoded here, should be retrieved from secure storage in production)
		if username != "admin" || password != "password" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			s.logger.Warn("Invalid credentials",
				zap.String("remoteAddr", r.RemoteAddr),
				zap.String("url", r.URL.RequestURI()))
			return
		}

		// Authentication passed, call the next handler
		next.ServeHTTP(w, r)
	})
}
