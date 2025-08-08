package server

import (
	"fmt"
	"log"
	"net/http"
)

// Server 文件下载服务器
type Server struct {
	root string // 文件根目录
	port int    // 服务端口
}

// NewServer 创建新的文件服务器
func NewServer(root string, port int) *Server {
	return &Server{
		root: root,
		port: port,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	fs := http.FileServer(http.Dir(s.root))

	handler := LoggingMiddleware(fs)

	http.Handle("/", handler)

	log.Printf("Serving %s on HTTP port %v\n", s.root, s.port)

	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, nil)
}
