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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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

	// Initialize OpenTelemetry with standard environment variable configuration
	tp, err := initTracer()
	if err != nil {
		logger.Error(context.Background(), "failed to initialize tracer", "error", err)
		// Continue without tracing rather than failing
	} else if tp != nil {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				logger.Error(context.Background(), "error shutting down tracer provider", "error", err)
			}
		}()
	}

	// Root context to allow coordinated cancellation (e.g., future signal handling)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM to cancel in-flight work
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigs
		logger.Warn(ctx, "received signal, cancelling", "signal", s.String())
		cancel()
	}()

	switch os.Args[1] {
	case "update":
		updateCmd.Parse(os.Args[2:])
		if err := runUpdate(ctx, *metadataDir, *checksumDir, *concurrency, *downloadConcurrency, !*noProgress, *maxRetries, *providerTimeout); err != nil {
			logger.Error(ctx, "update failed", "error", err)
			os.Exit(1)
		}
	case "validate":
		validateCmd.Parse(os.Args[2:])
		if err := runValidate(ctx, *validateMetadataDir, *validateConcurrency, *validateDelete); err != nil {
			logger.Error(ctx, "validation failed", "error", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// initTracer initializes the OpenTelemetry tracer provider using standard environment variables
// Configuration is controlled by OTEL_* environment variables:
// - OTEL_EXPORTER_OTLP_ENDPOINT: OTLP endpoint (default: http://localhost:4318)
// - OTEL_SERVICE_NAME: service name (default: java-metadata)
// - OTEL_TRACES_EXPORTER: exporter type (otlp, none, etc.)
func initTracer() (*sdktrace.TracerProvider, error) {
	// Check if tracing is disabled
	if os.Getenv("OTEL_TRACES_EXPORTER") == "none" {
		return nil, nil
	}

	// Create OTLP HTTP exporter (uses OTEL_EXPORTER_OTLP_ENDPOINT env var)
	exporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Get service name from environment or use default
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "java-metadata"
	}

	// Create resource with service name
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	return tp, nil
}

func runUpdate(parentCtx context.Context, metadataDir, checksumDir string, concurrency, downloadConcurrency int, showProgress bool, maxRetries int, providerTimeout time.Duration) error {
	startTime := time.Now()
	logger.Info(parentCtx, "starting update", "metadataDir", metadataDir, "checksumDir", checksumDir)

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
	logger.Info(parentCtx, "fetching releases from providers", "concurrency", concurrency)
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

			// Per-provider deadline derived from the shared parent context
			ctx, cancel := context.WithTimeout(parentCtx, providerTimeout)
			defer cancel()

			logger.Info(ctx, "fetching releases", "provider", p.Name())
			providerStart := time.Now()

			// Create tracing span for provider fetch
			tracer := otel.Tracer("java-metadata")
			ctx, span := tracer.Start(ctx, "provider.fetch_releases")
			span.SetAttributes(
				semconv.ServiceNameKey.String("java-metadata"),
			)
			defer span.End()

			metadata, err := p.FetchReleases(ctx)
			if err != nil {
				logger.Error(ctx, "failed to fetch releases", "provider", p.Name(), "error", err)
				errorChan <- fmt.Errorf("%s: %w", p.Name(), err)
				return
			}

			elapsed := time.Since(providerStart)
			logger.Info(ctx, "fetched releases", "provider", p.Name(), "count", len(metadata), "duration", elapsed)
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
		logger.Warn(parentCtx, "some providers failed", "errorCount", len(errors))
		logger.Warn(parentCtx, "failed providers:")
		for _, err := range errors {
			logger.Error(parentCtx, "  provider error", "error", err)
		}
		// Continue processing with data from successful providers
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
	logger.Info(parentCtx, "fetch complete", "totalReleases", len(allMetadata), "duration", fetchDuration)

	// Download artifacts and compute checksums
	logger.Info(parentCtx, "downloading artifacts and computing checksums", "concurrency", downloadConcurrency)
	downloadStart := time.Now()
	downloadedCount, skippedCount, failedCount, totalBytes, err := downloadAndComputeChecksums(
		allMetadata, metadataDir, checksumDir, downloadConcurrency, showProgress, maxRetries,
	)
	if err != nil {
		// Log error but continue - individual download failures are already logged
		logger.Warn(parentCtx, "some downloads encountered errors", "error", err)
	}
	downloadDuration := time.Since(downloadStart)
	logger.Info(parentCtx, "downloads complete",
		"downloaded", downloadedCount,
		"skipped", skippedCount,
		"failed", failedCount,
		"totalBytes", totalBytes,
		"duration", downloadDuration,
	)

	// Write vendor-specific metadata
	logger.Info(parentCtx, "writing vendor-specific metadata")
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
	logger.Info(parentCtx, "writing combined metadata")
	if err := output.WriteMetadataJSON(filepath.Join(metadataDir, "all.json"), allMetadata); err != nil {
		return fmt.Errorf("failed to write all.json: %w", err)
	}

	// Generate aggregated metadata structure
	logger.Info(parentCtx, "generating aggregated metadata structure")
	aggStart := time.Now()
	if err := output.AggregateMetadata(allMetadata, metadataDir); err != nil {
		return fmt.Errorf("failed to aggregate metadata: %w", err)
	}
	aggDuration := time.Since(aggStart)
	logger.Info(parentCtx, "aggregation complete", "duration", aggDuration)

	totalDuration := time.Since(startTime)

	// Provide detailed summary
	successfulProviders := len(allProviders) - len(errors)
	if len(errors) > 0 || failedCount > 0 {
		logger.Warn(parentCtx, "update completed with some failures",
			"totalDuration", totalDuration,
			"providersSucceeded", successfulProviders,
			"providersFailed", len(errors),
			"totalProviders", len(allProviders),
			"releasesProcessed", len(allMetadata),
			"downloadsSucceeded", downloadedCount,
			"downloadsFailed", failedCount,
			"downloadsSkipped", skippedCount,
		)
	} else {
		logger.Info(parentCtx, "update completed successfully",
			"totalDuration", totalDuration,
			"fetchDuration", fetchDuration,
			"downloadDuration", downloadDuration,
			"aggDuration", aggDuration,
			"providers", successfulProviders,
			"releases", len(allMetadata),
			"downloads", downloadedCount,
			"skipped", skippedCount,
		)
	}
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
			logger.Debug(context.Background(), "skipping existing file", "filename", m.Filename)
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
			if err := dl.DownloadFile(context.Background(), metadata.URL, archivePath); err != nil {
				logger.Error(context.Background(), "failed to download", "filename", metadata.Filename, "error", err)
				atomic.AddInt64(&failedCount, 1)
				return
			}

			// Compute checksums
			md5sum, sha1sum, sha256sum, sha512sum, err := downloader.ComputeChecksums(archivePath)
			if err != nil {
				logger.Error(context.Background(), "failed to compute checksums", "filename", metadata.Filename, "error", err)
				os.Remove(archivePath)
				atomic.AddInt64(&failedCount, 1)
				return
			}

			// Get file size
			size, err := downloader.FileSize(archivePath)
			if err != nil {
				logger.Error(context.Background(), "failed to get file size", "filename", metadata.Filename, "error", err)
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
				logger.Warn(context.Background(), "failed to write MD5 checksum", "filename", metadata.Filename, "error", err)
			}
			if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, metadata.Filename+".sha1"), sha1sum, metadata.Filename); err != nil {
				logger.Warn(context.Background(), "failed to write SHA1 checksum", "filename", metadata.Filename, "error", err)
			}
			if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, metadata.Filename+".sha256"), sha256sum, metadata.Filename); err != nil {
				logger.Warn(context.Background(), "failed to write SHA256 checksum", "filename", metadata.Filename, "error", err)
			}
			if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, metadata.Filename+".sha512"), sha512sum, metadata.Filename); err != nil {
				logger.Warn(context.Background(), "failed to write SHA512 checksum", "filename", metadata.Filename, "error", err)
			}

			// Remove the downloaded archive to save space
			os.Remove(archivePath)

			atomic.AddInt64(&downloadedCount, 1)
			atomic.AddInt64(&bytesDownloaded, size)
			logger.Debug(context.Background(), "download complete", "filename", metadata.Filename, "size", size)
		}(m, archivePath, metadataFile, vendorChecksumDir)
	}

	// Wait for all downloads to complete
	wg.Wait()

	return int(downloadedCount), int(skippedCount), int(failedCount), bytesDownloaded, nil
}

func runValidate(parentCtx context.Context, metadataDir string, concurrency int, deleteOnFailure bool) error {
	logger.Info(parentCtx, "starting validation", "metadataDir", metadataDir)
	if deleteOnFailure {
		logger.Warn(parentCtx, "delete mode enabled: files with failed URLs will be deleted")
	}

	// Read all.json file
	allJsonPath := filepath.Join(metadataDir, "all.json")
	data, err := os.ReadFile(allJsonPath)
	if err != nil {
		return fmt.Errorf("failed to read all.json: %w", err)
	}

	// Parse metadata list
	var allMetadata []models.Metadata
	if err := json.Unmarshal(data, &allMetadata); err != nil {
		return fmt.Errorf("failed to parse all.json: %w", err)
	}

	if len(allMetadata) == 0 {
		logger.Info(parentCtx, "no metadata entries found in all.json")
		return nil
	}

	logger.Info(parentCtx, "found metadata entries to validate", "count", len(allMetadata))

	// Create downloader for URL checking
	dl := downloader.NewDownloader(downloader.WithProgress(false))

	// Validate URLs concurrently
	var wg sync.WaitGroup
	var checked, failed int64
	semaphore := make(chan struct{}, concurrency)
	failedFilesChan := make(chan models.Metadata, len(allMetadata))

	startTime := time.Now()
	for _, metadata := range allMetadata {
		wg.Add(1)
		go func(m models.Metadata) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Exit early if cancelled
			if err := parentCtx.Err(); err != nil {
				return
			}

			// Check URL
			if err := dl.CheckURLExists(parentCtx, m.URL); err != nil {
				logger.Info(context.Background(), "URL not accessible", "filename", m.Filename, "url", m.URL, "error", err)
				failedFilesChan <- m
				atomic.AddInt64(&failed, 1)
			}

			atomic.AddInt64(&checked, 1)

			// Progress indicator
			if c := atomic.LoadInt64(&checked); c%100 == 0 {
				logger.Info(context.Background(), "validation progress", "checked", c, "total", len(allMetadata))
			}
		}(metadata)
	}

	// Wait for all validations to complete
	wg.Wait()
	close(failedFilesChan)

	// Collect failed entries
	var failedEntries []models.Metadata
	for m := range failedFilesChan {
		failedEntries = append(failedEntries, m)
	}

	duration := time.Since(startTime)

	// Print results
	logger.Info(parentCtx, "validation complete",
		"checked", checked,
		"failed", failed,
		"success", checked-failed,
		"duration", duration,
	)

	if len(failedEntries) > 0 {
		logger.Warn(parentCtx, "found inaccessible URLs", "count", len(failedEntries))
		for _, m := range failedEntries {
			logger.Warn(parentCtx, "inaccessible file", "filename", m.Filename, "vendor", m.Vendor)
		}

		// Delete files if requested
		if deleteOnFailure {
			logger.Info(parentCtx, "deleting failed entries", "count", len(failedEntries))
			checksumBase := filepath.Join(filepath.Dir(metadataDir), "checksums")
			var deletedChecksumCount, deleteFailedCount int

			for _, m := range failedEntries {
				// Delete checksum files using the explicit file paths from metadata
				checksumFiles := []string{m.MD5File, m.SHA1File, m.SHA256File, m.SHA512File}
				for _, checksumFile := range checksumFiles {
					if checksumFile == "" {
						continue
					}
					checksumPath := filepath.Join(checksumBase, m.Vendor, checksumFile)
					if err := os.Remove(checksumPath); err != nil {
						// Only log if file exists and deletion failed; ignore missing files
						if _, statErr := os.Stat(checksumPath); statErr == nil {
							logger.Error(parentCtx, "failed to delete checksum file", "file", checksumPath, "error", err)
							deleteFailedCount++
						}
					} else {
						deletedChecksumCount++
					}
				}
			}

			logger.Info(parentCtx, "deletion complete", "checksumsDeleted", deletedChecksumCount, "failed", deleteFailedCount)

			// Regenerate aggregated metadata files after deletions
			logger.Info(parentCtx, "regenerating aggregated metadata files")
			if err := regenerateAggregateMetadata(parentCtx, metadataDir, allMetadata); err != nil {
				logger.Error(parentCtx, "failed to regenerate aggregated metadata", "error", err)
				// Don't fail validation if aggregation fails - deletions were successful
			}
		}

		return fmt.Errorf("%d URLs are not accessible", len(failedEntries))
	}

	logger.Info(parentCtx, "all URLs are accessible")
	return nil
}

// regenerateAggregateMetadata reads all.json and regenerates aggregated metadata files
func regenerateAggregateMetadata(ctx context.Context, metadataDir string, allMetadata []models.Metadata) error {
	// Read all.json to ensure we have the latest state
	allJsonPath := filepath.Join(metadataDir, "all.json")
	data, err := os.ReadFile(allJsonPath)
	if err != nil {
		return fmt.Errorf("failed to read all.json: %w", err)
	}

	var currentMetadata models.MetadataList
	if err := json.Unmarshal(data, &currentMetadata); err != nil {
		return fmt.Errorf("failed to parse all.json: %w", err)
	}

	// Regenerate all aggregated files
	return output.AggregateMetadata(currentMetadata, metadataDir)
}
