package server

import (
	"fmt"
	"log"
	"net/http"
)

// Server file download server
type Server struct {
	root string // File root directory
	port int    // Service port
}

// NewServer creates a new file server
func NewServer(root string, port int) *Server {
	return &Server{
		root: root,
		port: port,
	}
}

// Start starts the server
func (s *Server) Start() error {
	fs := http.FileServer(http.Dir(s.root))

	handler := LoggingMiddleware(fs)

	// Create a new ServeMux to avoid conflicts with global DefaultServeMux
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	log.Printf("Serving %s on HTTP port %v\n", s.root, s.port)

	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, mux)
}
