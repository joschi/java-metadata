package downloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDownloader(t *testing.T) {
	d := NewDownloader()
	if d == nil {
		t.Fatal("NewDownloader returned nil")
	}
}

func TestComputeChecksums(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, World!")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	md5sum, sha1sum, sha256sum, sha512sum, err := ComputeChecksums(testFile)
	if err != nil {
		t.Fatalf("ComputeChecksums failed: %v", err)
	}

	// Verify that all checksums were computed
	if md5sum == "" {
		t.Error("MD5 checksum is empty")
	}
	if sha1sum == "" {
		t.Error("SHA1 checksum is empty")
	}
	if sha256sum == "" {
		t.Error("SHA256 checksum is empty")
	}
	if sha512sum == "" {
		t.Error("SHA512 checksum is empty")
	}

	// Verify known checksums for "Hello, World!"
	expectedMD5 := "65a8e27d8879283831b664bd8b7f0ad4"
	expectedSHA1 := "0a0a9f2a6772942557ab5355d76af442f8f65e01"
	expectedSHA256 := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"

	if md5sum != expectedMD5 {
		t.Errorf("MD5 mismatch: got %s, want %s", md5sum, expectedMD5)
	}
	if sha1sum != expectedSHA1 {
		t.Errorf("SHA1 mismatch: got %s, want %s", sha1sum, expectedSHA1)
	}
	if sha256sum != expectedSHA256 {
		t.Errorf("SHA256 mismatch: got %s, want %s", sha256sum, expectedSHA256)
	}
}

func TestWriteChecksumFile(t *testing.T) {
	tmpDir := t.TempDir()
	checksumFile := filepath.Join(tmpDir, "test.md5")

	err := WriteChecksumFile(checksumFile, "abc123", "test.tar.gz")
	if err != nil {
		t.Fatalf("WriteChecksumFile failed: %v", err)
	}

	// Read the file back
	content, err := os.ReadFile(checksumFile)
	if err != nil {
		t.Fatalf("Failed to read checksum file: %v", err)
	}

	expected := "abc123  test.tar.gz\n"
	if string(content) != expected {
		t.Errorf("Checksum file content mismatch: got %q, want %q", string(content), expected)
	}
}

func TestFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, World!")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err := FileSize(testFile)
	if err != nil {
		t.Fatalf("FileSize failed: %v", err)
	}

	expectedSize := int64(len(testContent))
	if size != expectedSize {
		t.Errorf("File size mismatch: got %d, want %d", size, expectedSize)
	}
}

func TestFileSizeNonExistent(t *testing.T) {
	_, err := FileSize("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestComputeChecksumsNonExistent(t *testing.T) {
	_, _, _, _, err := ComputeChecksums("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
