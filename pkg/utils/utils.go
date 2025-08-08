package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FormatBytes 格式化字节数为人类可读的格式
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration 格式化时间间隔
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// CalculateSpeed 计算传输速度
func CalculateSpeed(bytes int64, duration time.Duration) string {
	if duration == 0 {
		return "0 B/s"
	}
	speed := float64(bytes) / duration.Seconds()
	return FormatBytes(int64(speed)) + "/s"
}

// FileExists 检查文件是否存在
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// CreateFileWithDirs 创建文件，如果目录不存在则创建目录
func CreateFileWithDirs(filename string) (*os.File, error) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.Create(filename)
}

// GetFileSize 获取文件大小
func GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CalculateFileMD5 计算文件的MD5哈希值
func CalculateFileMD5(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// SanitizeFilename 清理文件名，移除不安全的字符
func SanitizeFilename(filename string) string {
	// 移除路径分隔符和其他不安全字符
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// ProgressBar 进度条结构
type ProgressBar struct {
	total   int64
	current int64
	width   int
	start   time.Time
}

// NewProgressBar 创建新的进度条
func NewProgressBar(total int64, width int) *ProgressBar {
	return &ProgressBar{
		total: total,
		width: width,
		start: time.Now(),
	}
}

// Update 更新进度
func (p *ProgressBar) Update(current int64) {
	p.current = current
}

// String 返回进度条的字符串表示
func (p *ProgressBar) String() string {
	if p.total == 0 {
		return "[未知大小]"
	}

	percent := float64(p.current) / float64(p.total) * 100
	filled := int(percent * float64(p.width) / 100)
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	
	elapsed := time.Since(p.start)
	speed := CalculateSpeed(p.current, elapsed)
	
	return fmt.Sprintf("[%s] %.1f%% %s/%s %s",
		bar,
		percent,
		FormatBytes(p.current),
		FormatBytes(p.total),
		speed)
}

// GetPercent 获取百分比
func (p *ProgressBar) GetPercent() float64 {
	if p.total == 0 {
		return 0
	}
	return float64(p.current) / float64(p.total) * 100
}

// IsComplete 检查是否完成
func (p *ProgressBar) IsComplete() bool {
	return p.current >= p.total
}

// ETAString 获取预计剩余时间
func (p *ProgressBar) ETAString() string {
	if p.current == 0 {
		return "计算中..."
	}

	elapsed := time.Since(p.start)
	rate := float64(p.current) / elapsed.Seconds()
	remaining := p.total - p.current
	
	if rate == 0 {
		return "未知"
	}

	eta := time.Duration(float64(remaining)/rate) * time.Second
	return FormatDuration(eta)
}