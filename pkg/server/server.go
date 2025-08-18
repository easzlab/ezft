package server

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Server file download server
type Server struct {
	root   string // File root directory
	port   int    // Service port
	logger *zap.Logger
}

// NewServer creates a new file server
func NewServer(root string, port int) *Server {
	return &Server{
		root: root,
		port: port,
	}
}

func (s *Server) SetLogger(logger *zap.Logger) {
	s.logger = logger
}

// Start starts the server
func (s *Server) Start() error {
	fs := http.FileServer(http.Dir(s.root))

	handler := s.LoggingMiddleware(fs)

	// Create a new ServeMux to avoid conflicts with global DefaultServeMux
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("Serving file server at %s, root: %s\n", addr, s.root)
	s.logger.Info("",
		zap.String("message", "Serving file server"),
		zap.String("root", s.root),
		zap.String("addr", addr),
	)

	return http.ListenAndServe(addr, mux)
}
