package downloader

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Downloader handles downloading files and computing checksums
type Downloader struct {
	client *http.Client
}

// NewDownloader creates a new Downloader instance
func NewDownloader() *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Minute, // Large files may take time
		},
	}
}

// DownloadFile downloads a file from the given URL to the specified path
func (d *Downloader) DownloadFile(url, outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	fmt.Printf("Downloading %s\n", url)

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: HTTP %d", url, resp.StatusCode)
	}

	// Create the output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer out.Close()

	// Copy the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}

// CheckURLExists checks if a URL is accessible using a HEAD request
func (d *Downloader) CheckURLExists(url string) error {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("URL not accessible: HTTP %d", resp.StatusCode)
	}

	return nil
}

// FileSize returns the size of a file in bytes
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ComputeChecksums computes MD5, SHA1, SHA256, and SHA512 checksums for a file
func ComputeChecksums(filePath string) (md5sum, sha1sum, sha256sum, sha512sum string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create hash instances
	md5Hash := md5.New()
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()
	sha512Hash := sha512.New()

	// Create a MultiWriter to compute all hashes in one pass
	multiWriter := io.MultiWriter(md5Hash, sha1Hash, sha256Hash, sha512Hash)

	// Read the file and compute hashes
	if _, err := io.Copy(multiWriter, file); err != nil {
		return "", "", "", "", fmt.Errorf("failed to compute hashes: %w", err)
	}

	// Convert hashes to hex strings
	md5sum = hex.EncodeToString(md5Hash.Sum(nil))
	sha1sum = hex.EncodeToString(sha1Hash.Sum(nil))
	sha256sum = hex.EncodeToString(sha256Hash.Sum(nil))
	sha512sum = hex.EncodeToString(sha512Hash.Sum(nil))

	return md5sum, sha1sum, sha256sum, sha512sum, nil
}

// ComputeChecksum computes a single checksum for a file using the specified algorithm
func ComputeChecksum(filePath, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var h hash.Hash
	switch algorithm {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteChecksumFile writes a checksum file in the format compatible with md5sum, sha1sum, etc.
// Format: <checksum>  <filename>
func WriteChecksumFile(checksumPath, checksum, filename string) error {
	dir := filepath.Dir(checksumPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	content := fmt.Sprintf("%s  %s\n", checksum, filename)
	if err := os.WriteFile(checksumPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write checksum file: %w", err)
	}

	return nil
}
