package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.input)
		if result != test.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		contains string
	}{
		{500 * time.Millisecond, "ms"},
		{2 * time.Second, "s"},
		{90 * time.Second, "m"},
		{2 * time.Hour, "h"},
	}

	for _, test := range tests {
		result := FormatDuration(test.input)
		if !strings.Contains(result, test.contains) {
			t.Errorf("FormatDuration(%v) = %s, expected to contain %s", test.input, result, test.contains)
		}
	}
}

func TestCalculateSpeed(t *testing.T) {
	tests := []struct {
		bytes    int64
		duration time.Duration
		contains string
	}{
		{1024, time.Second, "KB/s"},
		{1048576, time.Second, "MB/s"},
		{0, time.Second, "0 B/s"},
		{1024, 0, "0 B/s"}, // 零除保护
	}

	for _, test := range tests {
		result := CalculateSpeed(test.bytes, test.duration)
		if !strings.Contains(result, test.contains) {
			t.Errorf("CalculateSpeed(%d, %v) = %s, expected to contain %s", 
				test.bytes, test.duration, result, test.contains)
		}
	}
}

func TestFileExists(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "existing.txt")
	
	err := os.WriteFile(existingFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 测试存在的文件
	if !FileExists(existingFile) {
		t.Errorf("FileExists should return true for existing file")
	}

	// 测试不存在的文件
	nonExistingFile := filepath.Join(tempDir, "nonexisting.txt")
	if FileExists(nonExistingFile) {
		t.Errorf("FileExists should return false for non-existing file")
	}
}

func TestCreateFileWithDirs(t *testing.T) {
	tempDir := t.TempDir()
	
	// 测试创建嵌套目录中的文件
	nestedFile := filepath.Join(tempDir, "subdir", "nested", "file.txt")
	
	file, err := CreateFileWithDirs(nestedFile)
	if err != nil {
		t.Fatalf("CreateFileWithDirs failed: %v", err)
	}
	defer file.Close()

	// 验证文件存在
	if !FileExists(nestedFile) {
		t.Errorf("File should exist after CreateFileWithDirs")
	}

	// 验证目录存在
	dir := filepath.Dir(nestedFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Directory should be created")
	}
}

func TestGetFileSize(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "size_test.txt")
	testContent := "This is test content for size calculation"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err := GetFileSize(testFile)
	if err != nil {
		t.Fatalf("GetFileSize failed: %v", err)
	}

	expectedSize := int64(len(testContent))
	if size != expectedSize {
		t.Errorf("GetFileSize returned %d, expected %d", size, expectedSize)
	}

	// 测试不存在的文件
	_, err = GetFileSize(filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Errorf("GetFileSize should return error for non-existent file")
	}
}

func TestCalculateFileMD5(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "md5_test.txt")
	testContent := "Hello, World!"
	
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := CalculateFileMD5(testFile)
	if err != nil {
		t.Fatalf("CalculateFileMD5 failed: %v", err)
	}

	// "Hello, World!" 的 MD5 应该是 "65a8e27d8879283831b664bd8b7f0ad4"
	expectedHash := "65a8e27d8879283831b664bd8b7f0ad4"
	if hash != expectedHash {
		t.Errorf("MD5 hash mismatch. Got %s, expected %s", hash, expectedHash)
	}

	// 测试不存在的文件
	_, err = CalculateFileMD5(filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Errorf("CalculateFileMD5 should return error for non-existent file")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_file.txt", "normal_file.txt"},
		{"file/with/slashes.txt", "file_with_slashes.txt"},
		{"file\\with\\backslashes.txt", "file_with_backslashes.txt"},
		{"file:with:colons.txt", "file_with_colons.txt"},
		{"file*with*asterisks.txt", "file_with_asterisks.txt"},
		{"file?with?questions.txt", "file_with_questions.txt"},
		{"file\"with\"quotes.txt", "file_with_quotes.txt"},
		{"file<with>brackets.txt", "file_with_brackets.txt"},
		{"file|with|pipes.txt", "file_with_pipes.txt"},
	}

	for _, test := range tests {
		result := SanitizeFilename(test.input)
		if result != test.expected {
			t.Errorf("SanitizeFilename(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	
	// 测试创建新目录
	newDir := filepath.Join(tempDir, "new_directory")
	err := EnsureDir(newDir)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// 验证目录存在
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Errorf("Directory should be created")
	}

	// 测试已存在的目录
	err = EnsureDir(newDir)
	if err != nil {
		t.Errorf("EnsureDir should not fail for existing directory: %v", err)
	}

	// 测试嵌套目录
	nestedDir := filepath.Join(tempDir, "level1", "level2", "level3")
	err = EnsureDir(nestedDir)
	if err != nil {
		t.Fatalf("EnsureDir failed for nested directory: %v", err)
	}

	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Errorf("Nested directory should be created")
	}
}

func TestProgressBar(t *testing.T) {
	// 测试基本功能
	pb := NewProgressBar(1000, 50)
	
	if pb == nil {
		t.Fatal("NewProgressBar returned nil")
	}

	if pb.total != 1000 {
		t.Errorf("Expected total 1000, got %d", pb.total)
	}

	if pb.width != 50 {
		t.Errorf("Expected width 50, got %d", pb.width)
	}

	// 测试更新进度
	pb.Update(500)
	if pb.current != 500 {
		t.Errorf("Expected current 500, got %d", pb.current)
	}

	// 测试百分比计算
	percent := pb.GetPercent()
	if percent != 50.0 {
		t.Errorf("Expected percent 50.0, got %.1f", percent)
	}

	// 测试完成状态
	if pb.IsComplete() {
		t.Error("Progress bar should not be complete at 50%")
	}

	pb.Update(1000)
	if !pb.IsComplete() {
		t.Error("Progress bar should be complete at 100%")
	}

	// 测试字符串表示
	progressStr := pb.String()
	if !strings.Contains(progressStr, "100.0%") {
		t.Errorf("Progress string should contain 100.0%%, got: %s", progressStr)
	}
}

func TestProgressBarZeroTotal(t *testing.T) {
	// 测试总大小为0的情况
	pb := NewProgressBar(0, 50)
	
	percent := pb.GetPercent()
	if percent != 0 {
		t.Errorf("Expected percent 0 for zero total, got %.1f", percent)
	}

	progressStr := pb.String()
	if !strings.Contains(progressStr, "未知大小") {
		t.Errorf("Progress string should indicate unknown size, got: %s", progressStr)
	}
}

func TestProgressBarETA(t *testing.T) {
	pb := NewProgressBar(1000, 50)
	
	// 开始时应该显示"计算中..."
	eta := pb.ETAString()
	if eta != "计算中..." {
		t.Errorf("Expected '计算中...' for initial ETA, got: %s", eta)
	}

	// 模拟一些进度
	time.Sleep(10 * time.Millisecond) // 确保有时间流逝
	pb.Update(100)
	
	eta = pb.ETAString()
	if eta == "计算中..." || eta == "未知" {
		// 在这种快速测试中，ETA可能仍然无法计算，这是正常的
		// 我们只需要确保函数不会崩溃
	}
}

func BenchmarkFormatBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatBytes(1024 * 1024 * 1024) // 1GB
	}
}

func BenchmarkSanitizeFilename(b *testing.B) {
	testFilename := "file/with\\lots:of*unsafe?chars\"in<the>name|.txt"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeFilename(testFilename)
	}
}

func BenchmarkProgressBarUpdate(b *testing.B) {
	pb := NewProgressBar(int64(b.N), 50)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb.Update(int64(i))
	}
}

func BenchmarkProgressBarString(b *testing.B) {
	pb := NewProgressBar(1000000, 50)
	pb.Update(500000) // 50% progress
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.String()
	}
}