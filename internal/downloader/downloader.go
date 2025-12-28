package downloader

import (
	"context"
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

	"github.com/joschi/java-metadata/internal/logger"
	"github.com/schollz/progressbar/v3"
)

// Downloader handles downloading files and computing checksums
type Downloader struct {
	client       *http.Client
	maxRetries   int
	showProgress bool
}

// Option configures a Downloader
type Option func(*Downloader)

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(retries int) Option {
	return func(d *Downloader) {
		d.maxRetries = retries
	}
}

// WithProgress enables or disables progress bars
func WithProgress(show bool) Option {
	return func(d *Downloader) {
		d.showProgress = show
	}
}

// NewDownloader creates a new Downloader instance with optimized HTTP settings
func NewDownloader(opts ...Option) *Downloader {
	d := &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Minute, // Large files may take time
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   false,
			},
		},
		maxRetries:   3,
		showProgress: true,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// DownloadFile downloads a file from the given URL to the specified path with retry logic
func (d *Downloader) DownloadFile(url, outputPath string) error {
	var lastErr error

	for attempt := 1; attempt <= d.maxRetries; attempt++ {
		if attempt > 1 {
			backoff := time.Duration(1<<uint(attempt-2)) * time.Second
			logger.Debug("retrying download after backoff", "url", url, "attempt", attempt, "backoff", backoff)
			time.Sleep(backoff)
		}

		err := d.downloadOnce(url, outputPath)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on permanent errors (4xx status codes)
		if isPermanentError(err) {
			logger.Debug("permanent error, not retrying", "url", url, "error", err)
			break
		}

		if attempt < d.maxRetries {
			logger.Warn("download failed, will retry", "url", url, "attempt", attempt, "error", err)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", d.maxRetries, lastErr)
}

// downloadOnce performs a single download attempt
func (d *Downloader) downloadOnce(url, outputPath string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	logger.Info("downloading", "url", url)

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
		return &httpError{statusCode: resp.StatusCode, url: url}
	}

	// Create the output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer out.Close()

	// Copy the response body to the file with optional progress bar
	if d.showProgress && resp.ContentLength > 0 {
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			filepath.Base(outputPath),
		)
		defer bar.Close()

		// Wrap the response body with the progress bar
		proxyReader := progressbar.NewReader(resp.Body, bar)
		_, err = io.Copy(out, &proxyReader)
	} else {
		_, err = io.Copy(out, resp.Body)
	}
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}

// httpError represents an HTTP error with status code
type httpError struct {
	statusCode int
	url        string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d for %s", e.statusCode, e.url)
}

// isPermanentError returns true if the error is permanent (should not retry)
func isPermanentError(err error) bool {
	if httpErr, ok := err.(*httpError); ok {
		// 4xx errors are permanent (client errors)
		return httpErr.statusCode >= 400 && httpErr.statusCode < 500
	}
	return false
}

// CheckURLExists checks if a URL is accessible using a HEAD request
func (d *Downloader) CheckURLExists(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
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
