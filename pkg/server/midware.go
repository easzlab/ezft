package server

import (
	"log"
	"net/http"
	"time"
)

// responseWriter 包装原始的 ResponseWriter 以捕获响应信息
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

// 日志中间件
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 创建自定义的 ResponseWriter 来捕获响应信息
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     200, // 默认状态码
			responseSize:   0,
		}

		// 调用下一个处理器
		next.ServeHTTP(rw, r)

		// 获取请求信息
		duration := time.Since(start)
		userAgent := r.Header.Get("User-Agent")
		referer := r.Header.Get("Referer")
		contentLength := r.ContentLength
		if contentLength < 0 {
			contentLength = 0
		}

		// 记录详细日志
		log.Printf("[%s] %s %s %s - Status: %d - ReqSize: %d bytes - RespSize: %d bytes - Duration: %v - UserAgent: %q - Referer: %q",
			start.Format("2006-01-02 15:04:05"),
			r.RemoteAddr,
			r.Method,
			r.URL.RequestURI(), // 使用 RequestURI 包含查询参数
			rw.statusCode,
			contentLength,
			rw.responseSize,
			duration,
			userAgent,
			referer,
		)
	})
}

// 认证中间件
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从请求头中获取Basic Auth认证信息
		username, password, ok := r.BasicAuth()
		if !ok {
			// 如果没有提供认证信息，则要求认证
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 检查用户名和密码（这里使用硬编码，实际应用中应从安全存储中获取）
		if username != "admin" || password != "password" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// 认证通过，调用下一个处理器
		next.ServeHTTP(w, r)
	})
}
