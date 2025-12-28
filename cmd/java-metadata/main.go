package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/joschi/java-metadata/internal/downloader"
	"github.com/joschi/java-metadata/internal/logger"
	"github.com/joschi/java-metadata/internal/models"
	"github.com/joschi/java-metadata/internal/output"
	"github.com/joschi/java-metadata/internal/providers"
	"github.com/joschi/java-metadata/internal/providers/allproviders"
)

func main() {
	// Global flags
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	verbose := flag.Bool("verbose", false, "Enable verbose output (same as --log-level=debug)")
	quiet := flag.Bool("quiet", false, "Quiet mode (same as --log-level=error)")

	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	metadataDir := updateCmd.String("metadata-dir", "./docs/metadata", "Output directory for metadata")
	checksumDir := updateCmd.String("checksum-dir", "./docs/checksums", "Output directory for checksums")
	concurrency := updateCmd.Int("concurrency", 4, "Number of concurrent provider fetches")
	downloadConcurrency := updateCmd.Int("download-concurrency", 3, "Number of concurrent downloads")
	noProgress := updateCmd.Bool("no-progress", false, "Disable progress bars")
	maxRetries := updateCmd.Int("max-retries", 3, "Maximum number of retry attempts for downloads")
	providerTimeout := updateCmd.Duration("provider-timeout", 5*time.Minute, "Per-provider timeout (e.g. 2m, 30s)")

	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	validateMetadataDir := validateCmd.String("metadata-dir", "./docs/metadata", "Directory containing metadata files")
	validateConcurrency := validateCmd.Int("concurrency", 10, "Number of concurrent URL checks")
	validateDelete := validateCmd.Bool("delete", false, "Delete files that fail validation")

	if len(os.Args) < 2 {
		fmt.Println("Usage: java-metadata [global-options] <command> [options]")
		fmt.Println("\nGlobal Options:")
		flag.PrintDefaults()
		fmt.Println("\nCommands:")
		fmt.Println("  update      Fetch and update metadata for all vendors")
		fmt.Println("  validate    Validate URLs in metadata files")
		os.Exit(1)
	}

	// Parse global flags
	flag.Parse()

	// Configure logging
	level := logger.ParseLevel(*logLevel)
	if *verbose {
		level = logger.LevelDebug
	}
	if *quiet {
		level = logger.LevelError
	}
	logger.SetLevel(level)

	// Root context to allow coordinated cancellation (e.g., future signal handling)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM to cancel in-flight work
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigs
		logger.Warn("received signal, cancelling", "signal", s.String())
		cancel()
	}()

	switch os.Args[1] {
	case "update":
		updateCmd.Parse(os.Args[2:])
		if err := runUpdate(ctx, *metadataDir, *checksumDir, *concurrency, *downloadConcurrency, !*noProgress, *maxRetries, *providerTimeout); err != nil {
			logger.Error("update failed", "error", err)
			os.Exit(1)
		}
	case "validate":
		validateCmd.Parse(os.Args[2:])
		if err := runValidate(ctx, *validateMetadataDir, *validateConcurrency, *validateDelete); err != nil {
			logger.Error("validation failed", "error", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runUpdate(parentCtx context.Context, metadataDir, checksumDir string, concurrency, downloadConcurrency int, showProgress bool, maxRetries int, providerTimeout time.Duration) error {
	startTime := time.Now()
	logger.Info("starting update", "metadataDir", metadataDir, "checksumDir", checksumDir)

	// Create registry and register all providers
	registry := providers.NewRegistry()
	allproviders.RegisterAll(registry)

	// Create output directories
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}
	if err := os.MkdirAll(checksumDir, 0755); err != nil {
		return fmt.Errorf("failed to create checksum directory: %w", err)
	}

	// Fetch metadata from all providers concurrently
	logger.Info("fetching releases from providers", "concurrency", concurrency)
	allProviders := registry.All()
	var wg sync.WaitGroup
	metadataChan := make(chan []models.Metadata, len(allProviders))
	errorChan := make(chan error, len(allProviders))
	semaphore := make(chan struct{}, concurrency)

	fetchStart := time.Now()
	for _, provider := range allProviders {
		wg.Add(1)
		go func(p providers.Provider) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			logger.Info("fetching releases", "provider", p.Name())
			providerStart := time.Now()
			// Per-provider deadline derived from the shared parent context
			ctx, cancel := context.WithTimeout(parentCtx, providerTimeout)
			defer cancel()
			metadata, err := p.FetchReleases(ctx)
			if err != nil {
				logger.Error("failed to fetch releases", "provider", p.Name(), "error", err)
				errorChan <- fmt.Errorf("%s: %w", p.Name(), err)
				return
			}

			elapsed := time.Since(providerStart)
			logger.Info("fetched releases", "provider", p.Name(), "count", len(metadata), "duration", elapsed)
			metadataChan <- metadata
		}(provider)
	}

	// Wait for all providers to finish
	go func() {
		wg.Wait()
		close(metadataChan)
		close(errorChan)
	}()

	// Collect errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		logger.Warn("some providers failed", "errorCount", len(errors))
		for _, err := range errors {
			logger.Error("provider error", "error", err)
		}
		return fmt.Errorf("failed to fetch releases from %d providers", len(errors))
	}

	// Combine all metadata
	var allMetadata []models.Metadata
	vendorMetadata := make(map[string][]models.Metadata)

	for metadata := range metadataChan {
		allMetadata = append(allMetadata, metadata...)
		for _, m := range metadata {
			vendorMetadata[m.Vendor] = append(vendorMetadata[m.Vendor], m)
		}
	}

	fetchDuration := time.Since(fetchStart)
	logger.Info("fetch complete", "totalReleases", len(allMetadata), "duration", fetchDuration)

	// Download artifacts and compute checksums
	logger.Info("downloading artifacts and computing checksums", "concurrency", downloadConcurrency)
	downloadStart := time.Now()
	downloadedCount, skippedCount, failedCount, totalBytes, err := downloadAndComputeChecksums(
		allMetadata, metadataDir, checksumDir, downloadConcurrency, showProgress, maxRetries,
	)
	if err != nil {
		return fmt.Errorf("failed to download artifacts: %w", err)
	}
	downloadDuration := time.Since(downloadStart)
	logger.Info("downloads complete",
		"downloaded", downloadedCount,
		"skipped", skippedCount,
		"failed", failedCount,
		"totalBytes", totalBytes,
		"duration", downloadDuration,
	)

	// Write vendor-specific metadata
	logger.Info("writing vendor-specific metadata")
	for vendor, metadata := range vendorMetadata {
		vendorDir := filepath.Join(metadataDir, "vendor", vendor)
		if err := os.MkdirAll(vendorDir, 0755); err != nil {
			return fmt.Errorf("failed to create vendor directory: %w", err)
		}

		// Write all.json for this vendor
		if err := output.WriteMetadataJSON(filepath.Join(vendorDir, "all.json"), metadata); err != nil {
			return fmt.Errorf("failed to write vendor metadata: %w", err)
		}

		// Write individual metadata files
		for _, m := range metadata {
			metadataFile := filepath.Join(vendorDir, m.Filename+".json")
			if err := output.WriteSingleMetadataJSON(metadataFile, m); err != nil {
				return fmt.Errorf("failed to write metadata file: %w", err)
			}
		}
	}

	// Write combined all.json
	logger.Info("writing combined metadata")
	if err := output.WriteMetadataJSON(filepath.Join(metadataDir, "all.json"), allMetadata); err != nil {
		return fmt.Errorf("failed to write all.json: %w", err)
	}

	// Generate aggregated metadata structure
	logger.Info("generating aggregated metadata structure")
	aggStart := time.Now()
	if err := output.AggregateMetadata(allMetadata, metadataDir); err != nil {
		return fmt.Errorf("failed to aggregate metadata: %w", err)
	}
	aggDuration := time.Since(aggStart)
	logger.Info("aggregation complete", "duration", aggDuration)

	totalDuration := time.Since(startTime)
	logger.Info("update completed successfully",
		"totalDuration", totalDuration,
		"fetchDuration", fetchDuration,
		"downloadDuration", downloadDuration,
		"aggDuration", aggDuration,
	)
	return nil
}

func downloadAndComputeChecksums(metadata []models.Metadata, metadataDir, checksumDir string, concurrency int, showProgress bool, maxRetries int) (downloaded, skipped, failed int, totalBytes int64, err error) {
	// Create downloader with options
	dl := downloader.NewDownloader(
		downloader.WithProgress(showProgress),
		downloader.WithMaxRetries(maxRetries),
	)

	// Counters
	var downloadedCount, skippedCount, failedCount int64
	var bytesDownloaded int64

	// Worker pool for concurrent downloads
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	var metadataMu sync.Mutex // Protects metadata slice updates

	for i := range metadata {
		m := &metadata[i]

		// Construct paths
		vendorMetadataDir := filepath.Join(metadataDir, "vendor", m.Vendor)
		vendorChecksumDir := filepath.Join(checksumDir, m.Vendor)
		archivePath := filepath.Join(vendorMetadataDir, m.Filename)
		metadataFile := filepath.Join(vendorMetadataDir, m.Filename+".json")

		// Skip if metadata file already exists
		if _, err := os.Stat(metadataFile); err == nil {
			logger.Debug("skipping existing file", "filename", m.Filename)
			atomic.AddInt64(&skippedCount, 1)
			continue
		}

		wg.Add(1)
		go func(metadata *models.Metadata, archivePath, metadataFile, vendorChecksumDir string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Download the artifact
			if err := dl.DownloadFile(metadata.URL, archivePath); err != nil {
				logger.Error("failed to download", "filename", metadata.Filename, "error", err)
				atomic.AddInt64(&failedCount, 1)
				return
			}

			// Compute checksums
			md5sum, sha1sum, sha256sum, sha512sum, err := downloader.ComputeChecksums(archivePath)
			if err != nil {
				logger.Error("failed to compute checksums", "filename", metadata.Filename, "error", err)
				os.Remove(archivePath)
				atomic.AddInt64(&failedCount, 1)
				return
			}

			// Get file size
			size, err := downloader.FileSize(archivePath)
			if err != nil {
				logger.Error("failed to get file size", "filename", metadata.Filename, "error", err)
				os.Remove(archivePath)
				atomic.AddInt64(&failedCount, 1)
				return
			}

			// Update metadata (needs mutex since we're updating slice element)
			metadataMu.Lock()
			metadata.MD5 = md5sum
			metadata.SHA1 = sha1sum
			metadata.SHA256 = sha256sum
			metadata.SHA512 = sha512sum
			metadata.Size = size
			metadataMu.Unlock()

			// Write checksum files
			if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, metadata.Filename+".md5"), md5sum, metadata.Filename); err != nil {
				logger.Warn("failed to write MD5 checksum", "filename", metadata.Filename, "error", err)
			}
			if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, metadata.Filename+".sha1"), sha1sum, metadata.Filename); err != nil {
				logger.Warn("failed to write SHA1 checksum", "filename", metadata.Filename, "error", err)
			}
			if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, metadata.Filename+".sha256"), sha256sum, metadata.Filename); err != nil {
				logger.Warn("failed to write SHA256 checksum", "filename", metadata.Filename, "error", err)
			}
			if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, metadata.Filename+".sha512"), sha512sum, metadata.Filename); err != nil {
				logger.Warn("failed to write SHA512 checksum", "filename", metadata.Filename, "error", err)
			}

			// Remove the downloaded archive to save space
			os.Remove(archivePath)

			atomic.AddInt64(&downloadedCount, 1)
			atomic.AddInt64(&bytesDownloaded, size)
			logger.Debug("download complete", "filename", metadata.Filename, "size", size)
		}(m, archivePath, metadataFile, vendorChecksumDir)
	}

	// Wait for all downloads to complete
	wg.Wait()

	return int(downloadedCount), int(skippedCount), int(failedCount), bytesDownloaded, nil
}

func runValidate(parentCtx context.Context, metadataDir string, concurrency int, deleteOnFailure bool) error {
	logger.Info("starting validation", "metadataDir", metadataDir)
	if deleteOnFailure {
		logger.Warn("delete mode enabled: files with failed URLs will be deleted")
	}

	// Find all metadata JSON files
	var metadataFiles []string
	err := filepath.Walk(metadataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Respect cancellation during directory walk
		if cerr := parentCtx.Err(); cerr != nil {
			return cerr
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" && filepath.Base(path) != "all.json" {
			metadataFiles = append(metadataFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk metadata directory: %w", err)
	}

	if len(metadataFiles) == 0 {
		logger.Info("no metadata files found")
		return nil
	}

	logger.Info("found metadata files to validate", "count", len(metadataFiles))

	// Create downloader for URL checking
	dl := downloader.NewDownloader(downloader.WithProgress(false))

	// Validate URLs concurrently
	var wg sync.WaitGroup
	var checked, failed int64
	semaphore := make(chan struct{}, concurrency)
	failedFilesChan := make(chan string, len(metadataFiles))

	startTime := time.Now()
	for _, file := range metadataFiles {
		wg.Add(1)
		go func(metadataFile string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Exit early if cancelled
			if err := parentCtx.Err(); err != nil {
				return
			}

			// Read metadata file
			data, err := os.ReadFile(metadataFile)
			if err != nil {
				logger.Error("failed to read metadata file", "file", metadataFile, "error", err)
				atomic.AddInt64(&failed, 1)
				return
			}

			// Parse metadata
			var metadata models.Metadata
			if err := json.Unmarshal(data, &metadata); err != nil {
				logger.Error("failed to parse metadata file", "file", metadataFile, "error", err)
				atomic.AddInt64(&failed, 1)
				return
			}

			// Check URL
			if err := parentCtx.Err(); err != nil {
				return
			}

			if err := dl.CheckURLExists(parentCtx, metadata.URL); err != nil {
				logger.Debug("URL not accessible", "file", metadataFile, "url", metadata.URL, "error", err)
				failedFilesChan <- metadataFile
				atomic.AddInt64(&failed, 1)
			}

			atomic.AddInt64(&checked, 1)

			// Progress indicator
			if c := atomic.LoadInt64(&checked); c%100 == 0 {
				logger.Info("validation progress", "checked", c, "total", len(metadataFiles))
			}
		}(file)
	}

	// Wait for all validations to complete
	wg.Wait()
	close(failedFilesChan)

	// Collect failed files
	var failedFiles []string
	for file := range failedFilesChan {
		failedFiles = append(failedFiles, file)
	}

	duration := time.Since(startTime)

	// Print results
	logger.Info("validation complete",
		"checked", checked,
		"failed", failed,
		"success", checked-failed,
		"duration", duration,
	)

	if len(failedFiles) > 0 {
		logger.Warn("found inaccessible URLs", "count", len(failedFiles))
		for _, file := range failedFiles {
			logger.Warn("inaccessible file", "path", file)
		}

		// Delete files if requested
		if deleteOnFailure {
			logger.Info("deleting failed files", "count", len(failedFiles))
			var deletedCount, deleteFailedCount int
			for _, file := range failedFiles {
				if err := os.Remove(file); err != nil {
					logger.Error("failed to delete file", "file", file, "error", err)
					deleteFailedCount++
				} else {
					deletedCount++
				}
			}
			logger.Info("deletion complete", "deleted", deletedCount, "failed", deleteFailedCount)
		}

		return fmt.Errorf("%d URLs are not accessible", len(failedFiles))
	}

	logger.Info("all URLs are accessible")
	return nil
}
