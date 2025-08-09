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
		{1024, 0, "0 B/s"}, // Zero division protection
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
	// Create temporary file
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "existing.txt")

	err := os.WriteFile(existingFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	if !FileExists(existingFile) {
		t.Errorf("FileExists should return true for existing file")
	}

	// Test non-existing file
	nonExistingFile := filepath.Join(tempDir, "nonexisting.txt")
	if FileExists(nonExistingFile) {
		t.Errorf("FileExists should return false for non-existing file")
	}
}

func TestCreateFileWithDirs(t *testing.T) {
	tempDir := t.TempDir()

	// Test creating file in nested directories
	nestedFile := filepath.Join(tempDir, "subdir", "nested", "file.txt")

	file, err := CreateFileWithDirs(nestedFile)
	if err != nil {
		t.Fatalf("CreateFileWithDirs failed: %v", err)
	}
	defer file.Close()

	// Verify file exists
	if !FileExists(nestedFile) {
		t.Errorf("File should exist after CreateFileWithDirs")
	}

	// Verify directory exists
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

	// Test non-existing file
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

	// MD5 of "Hello, World!" should be "65a8e27d8879283831b664bd8b7f0ad4"
	expectedHash := "65a8e27d8879283831b664bd8b7f0ad4"
	if hash != expectedHash {
		t.Errorf("MD5 hash mismatch. Got %s, expected %s", hash, expectedHash)
	}

	// Test non-existing file
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

	// Test creating new directory
	newDir := filepath.Join(tempDir, "new_directory")
	err := EnsureDir(newDir)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Errorf("Directory should be created")
	}

	// Test existing directory
	err = EnsureDir(newDir)
	if err != nil {
		t.Errorf("EnsureDir should not fail for existing directory: %v", err)
	}

	// Test nested directories
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
	// Test basic functionality
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

	// Test progress update
	pb.Update(500)
	if pb.current != 500 {
		t.Errorf("Expected current 500, got %d", pb.current)
	}
}
