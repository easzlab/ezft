package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	http.HandleFunc("/download/", s.handleDownload)
	http.HandleFunc("/info/", s.handleFileInfo)
	http.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("文件下载服务器启动在端口 %d，根目录: %s", s.port, s.root)
	return http.ListenAndServe(addr, nil)
}

// handleDownload 处理文件下载请求，支持 Range 请求实现断点续传
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	// 提取文件路径
	filePath := strings.TrimPrefix(r.URL.Path, "/download/")
	if filePath == "" {
		http.Error(w, "文件路径不能为空", http.StatusBadRequest)
		return
	}

	// 构建完整文件路径
	fullPath := filepath.Join(s.root, filePath)
	
	// 安全检查：防止路径遍历攻击
	if !strings.HasPrefix(fullPath, s.root) {
		http.Error(w, "无效的文件路径", http.StatusForbidden)
		return
	}

	// 检查文件是否存在
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "文件不存在", http.StatusNotFound)
		} else {
			http.Error(w, "文件访问错误", http.StatusInternalServerError)
		}
		return
	}

	// 检查是否为文件（不是目录）
	if fileInfo.IsDir() {
		http.Error(w, "不能下载目录", http.StatusBadRequest)
		return
	}

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "文件打开失败", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fileSize := fileInfo.Size()
	
	// 设置基本响应头
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filePath)))
	w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))

	// 处理 Range 请求（断点续传）
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// 没有 Range 请求，返回完整文件
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, file)
		return
	}

	// 解析 Range 头
	ranges, err := parseRange(rangeHeader, fileSize)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		http.Error(w, "无效的 Range 请求", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if len(ranges) != 1 {
		http.Error(w, "不支持多段 Range 请求", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// 处理单段 Range 请求
	start, end := ranges[0].start, ranges[0].end
	contentLength := end - start + 1

	// 设置 Range 响应头
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.WriteHeader(http.StatusPartialContent)

	// 定位到起始位置
	_, err = file.Seek(start, io.SeekStart)
	if err != nil {
		http.Error(w, "文件定位失败", http.StatusInternalServerError)
		return
	}

	// 发送指定范围的数据
	io.CopyN(w, file, contentLength)
}

// handleFileInfo 处理文件信息请求
func (s *Server) handleFileInfo(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.Path, "/info/")
	if filePath == "" {
		http.Error(w, "文件路径不能为空", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(s.root, filePath)
	
	// 安全检查
	if !strings.HasPrefix(fullPath, s.root) {
		http.Error(w, "无效的文件路径", http.StatusForbidden)
		return
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "文件不存在", http.StatusNotFound)
		} else {
			http.Error(w, "文件访问错误", http.StatusInternalServerError)
		}
		return
	}

	if fileInfo.IsDir() {
		http.Error(w, "不能获取目录信息", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"name":"%s","size":%d,"modified":"%s"}`,
		fileInfo.Name(),
		fileInfo.Size(),
		fileInfo.ModTime().Format(time.RFC3339))
}

// handleHealth 健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// Range 表示字节范围
type Range struct {
	start, end int64
}

// parseRange 解析 Range 头
func parseRange(rangeHeader string, size int64) ([]Range, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, fmt.Errorf("invalid range header")
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	ranges := strings.Split(rangeSpec, ",")
	
	var result []Range
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		if strings.Contains(r, "-") {
			parts := strings.Split(r, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid range format")
			}

			var start, end int64
			var err error

			if parts[0] == "" {
				// 后缀范围，如 "-500"
				if parts[1] == "" {
					return nil, fmt.Errorf("invalid suffix range")
				}
				suffixLength, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return nil, err
				}
				start = size - suffixLength
				if start < 0 {
					start = 0
				}
				end = size - 1
			} else if parts[1] == "" {
				// 前缀范围，如 "500-"
				start, err = strconv.ParseInt(parts[0], 10, 64)
				if err != nil {
					return nil, err
				}
				end = size - 1
			} else {
				// 完整范围，如 "500-999"
				start, err = strconv.ParseInt(parts[0], 10, 64)
				if err != nil {
					return nil, err
				}
				end, err = strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return nil, err
				}
			}

			// 验证范围有效性
			if start < 0 || end >= size || start > end {
				return nil, fmt.Errorf("invalid range values")
			}

			result = append(result, Range{start: start, end: end})
		}
	}

	return result, nil
}