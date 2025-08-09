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

// FormatBytes formats bytes to human readable format
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

// FormatDuration formats time duration
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

// CalculateSpeed calculates transfer speed
func CalculateSpeed(bytes int64, duration time.Duration) string {
	if duration == 0 {
		return "0 B/s"
	}
	speed := float64(bytes) / duration.Seconds()
	return FormatBytes(int64(speed)) + "/s"
}

// FileExists checks if file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// CreateFileWithDirs creates file, create directories if they don't exist
func CreateFileWithDirs(filename string) (*os.File, error) {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.Create(filename)
}

// GetFileSize gets file size
func GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CalculateFileMD5 calculates MD5 hash of file
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

// SanitizeFilename cleans filename, removes unsafe characters
func SanitizeFilename(filename string) string {
	// Remove path separators and other unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// EnsureDir ensures directory exists, create if it doesn't exist
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// ProgressBar progress bar structure
type ProgressBar struct {
	total   int64
	current int64
	width   int
	start   time.Time
}

// NewProgressBar creates new progress bar
func NewProgressBar(total int64, width int) *ProgressBar {
	return &ProgressBar{
		total: total,
		width: width,
		start: time.Now(),
	}
}

// Update updates progress
func (p *ProgressBar) Update(current int64) {
	p.current = current
}

// String returns string representation of progress bar
func (p *ProgressBar) String() string {
	if p.total == 0 {
		return "[Unknown size]"
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

// GetPercent gets percentage
func (p *ProgressBar) GetPercent() float64 {
	if p.total == 0 {
		return 0
	}
	return float64(p.current) / float64(p.total) * 100
}

// IsComplete checks if complete
func (p *ProgressBar) IsComplete() bool {
	return p.current >= p.total
}

// ETAString gets estimated remaining time
func (p *ProgressBar) ETAString() string {
	if p.current == 0 {
		return "Calculating..."
	}

	elapsed := time.Since(p.start)
	rate := float64(p.current) / elapsed.Seconds()
	remaining := p.total - p.current

	if rate == 0 {
		return "Unknown"
	}

	eta := time.Duration(float64(remaining)/rate) * time.Second
	return FormatDuration(eta)
}
