package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/joschi/java-metadata/internal/downloader"
	"github.com/joschi/java-metadata/internal/models"
	"github.com/joschi/java-metadata/internal/output"
	"github.com/joschi/java-metadata/internal/providers"
	"github.com/joschi/java-metadata/internal/providers/allproviders"
)

func main() {
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	metadataDir := updateCmd.String("metadata-dir", "./docs/metadata", "Output directory for metadata")
	checksumDir := updateCmd.String("checksum-dir", "./docs/checksums", "Output directory for checksums")
	concurrency := updateCmd.Int("concurrency", 4, "Number of concurrent provider fetches")

	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	validateMetadataDir := validateCmd.String("metadata-dir", "./docs/metadata", "Directory containing metadata files")
	validateConcurrency := validateCmd.Int("concurrency", 10, "Number of concurrent URL checks")
	validateDelete := validateCmd.Bool("delete", false, "Delete files that fail validation")

	if len(os.Args) < 2 {
		fmt.Println("Usage: java-metadata <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  update      Fetch and update metadata for all vendors")
		fmt.Println("  validate    Validate URLs in metadata files")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "update":
		updateCmd.Parse(os.Args[2:])
		if err := runUpdate(*metadataDir, *checksumDir, *concurrency); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		validateCmd.Parse(os.Args[2:])
		if err := runValidate(*validateMetadataDir, *validateConcurrency, *validateDelete); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runUpdate(metadataDir, checksumDir string, concurrency int) error {
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
	allProviders := registry.All()
	var wg sync.WaitGroup
	metadataChan := make(chan []models.Metadata, len(allProviders))
	errorChan := make(chan error, len(allProviders))
	semaphore := make(chan struct{}, concurrency)

	for _, provider := range allProviders {
		wg.Add(1)
		go func(p providers.Provider) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("Fetching releases from %s...\n", p.Name())
			metadata, err := p.FetchReleases()
			if err != nil {
				errorChan <- fmt.Errorf("%s: %w", p.Name(), err)
				return
			}

			fmt.Printf("Fetched %d releases from %s\n", len(metadata), p.Name())
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
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return fmt.Errorf("failed to fetch releases from some providers")
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

	fmt.Printf("Total releases collected: %d\n", len(allMetadata))

	// Download artifacts and compute checksums
	fmt.Println("Downloading artifacts and computing checksums...")
	if err := downloadAndComputeChecksums(allMetadata, metadataDir, checksumDir); err != nil {
		return fmt.Errorf("failed to download artifacts: %w", err)
	}

	// Write vendor-specific metadata
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
	fmt.Println("Writing combined metadata...")
	if err := output.WriteMetadataJSON(filepath.Join(metadataDir, "all.json"), allMetadata); err != nil {
		return fmt.Errorf("failed to write all.json: %w", err)
	}

	// Generate aggregated metadata structure
	fmt.Println("Generating aggregated metadata structure...")
	if err := output.AggregateMetadata(allMetadata, metadataDir); err != nil {
		return fmt.Errorf("failed to aggregate metadata: %w", err)
	}

	fmt.Println("Update completed successfully!")
	return nil
}

func downloadAndComputeChecksums(metadata []models.Metadata, metadataDir, checksumDir string) error {
	dl := downloader.NewDownloader()

	for i := range metadata {
		m := &metadata[i]

		// Construct paths
		vendorMetadataDir := filepath.Join(metadataDir, "vendor", m.Vendor)
		vendorChecksumDir := filepath.Join(checksumDir, m.Vendor)
		archivePath := filepath.Join(vendorMetadataDir, m.Filename)
		metadataFile := filepath.Join(vendorMetadataDir, m.Filename+".json")

		// Skip if metadata file already exists
		if _, err := os.Stat(metadataFile); err == nil {
			fmt.Printf("Skipping %s (already exists)\n", m.Filename)
			continue
		}

		// Download the artifact
		if err := dl.DownloadFile(m.URL, archivePath); err != nil {
			fmt.Printf("Failed to download %s: %v\n", m.Filename, err)
			continue
		}

		// Compute checksums
		md5sum, sha1sum, sha256sum, sha512sum, err := downloader.ComputeChecksums(archivePath)
		if err != nil {
			fmt.Printf("Failed to compute checksums for %s: %v\n", m.Filename, err)
			os.Remove(archivePath)
			continue
		}

		// Update metadata
		m.MD5 = md5sum
		m.SHA1 = sha1sum
		m.SHA256 = sha256sum
		m.SHA512 = sha512sum

		// Get file size
		size, err := downloader.FileSize(archivePath)
		if err != nil {
			fmt.Printf("Failed to get size for %s: %v\n", m.Filename, err)
			os.Remove(archivePath)
			continue
		}
		m.Size = size

		// Write checksum files
		if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, m.Filename+".md5"), md5sum, m.Filename); err != nil {
			fmt.Printf("Failed to write MD5 checksum for %s: %v\n", m.Filename, err)
		}
		if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, m.Filename+".sha1"), sha1sum, m.Filename); err != nil {
			fmt.Printf("Failed to write SHA1 checksum for %s: %v\n", m.Filename, err)
		}
		if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, m.Filename+".sha256"), sha256sum, m.Filename); err != nil {
			fmt.Printf("Failed to write SHA256 checksum for %s: %v\n", m.Filename, err)
		}
		if err := downloader.WriteChecksumFile(filepath.Join(vendorChecksumDir, m.Filename+".sha512"), sha512sum, m.Filename); err != nil {
			fmt.Printf("Failed to write SHA512 checksum for %s: %v\n", m.Filename, err)
		}

		// Remove the downloaded archive to save space
		os.Remove(archivePath)
	}

	return nil
}

func runValidate(metadataDir string, concurrency int, deleteOnFailure bool) error {
	fmt.Printf("Validating URLs in metadata files from %s\n", metadataDir)
	if deleteOnFailure {
		fmt.Println("⚠️  Delete mode enabled: Files with failed URLs will be deleted")
	}

	// Find all metadata JSON files
	var metadataFiles []string
	err := filepath.Walk(metadataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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
		fmt.Println("No metadata files found")
		return nil
	}

	fmt.Printf("Found %d metadata files to validate\n", len(metadataFiles))

	// Create downloader for URL checking
	dl := downloader.NewDownloader()

	// Validate URLs concurrently
	var wg sync.WaitGroup
	var checked, failed int64
	semaphore := make(chan struct{}, concurrency)
	failedFilesChan := make(chan string, len(metadataFiles))

	for _, file := range metadataFiles {
		wg.Add(1)
		go func(metadataFile string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Read metadata file
			data, err := os.ReadFile(metadataFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", metadataFile, err)
				atomic.AddInt64(&failed, 1)
				return
			}

			// Parse metadata
			var metadata models.Metadata
			if err := json.Unmarshal(data, &metadata); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to parse %s: %v\n", metadataFile, err)
				atomic.AddInt64(&failed, 1)
				return
			}

			// Check URL
			if err := dl.CheckURLExists(metadata.URL); err != nil {
				failedFilesChan <- metadataFile
				atomic.AddInt64(&failed, 1)
			}

			atomic.AddInt64(&checked, 1)

			// Progress indicator
			if c := atomic.LoadInt64(&checked); c%100 == 0 {
				fmt.Printf("Checked %d/%d URLs...\n", c, len(metadataFiles))
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

	// Print results
	fmt.Printf("\nValidation complete:\n")
	fmt.Printf("  Checked: %d\n", checked)
	fmt.Printf("  Failed:  %d\n", failed)
	fmt.Printf("  Success: %d\n", checked-failed)

	if len(failedFiles) > 0 {
		fmt.Println("\nFailed files:")
		for _, file := range failedFiles {
			fmt.Println(file)
		}

		// Delete files if requested
		if deleteOnFailure {
			fmt.Printf("\nDeleting %d failed files...\n", len(failedFiles))
			var deletedCount, deleteFailedCount int
			for _, file := range failedFiles {
				if err := os.Remove(file); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", file, err)
					deleteFailedCount++
				} else {
					deletedCount++
				}
			}
			fmt.Printf("Deleted: %d files\n", deletedCount)
			if deleteFailedCount > 0 {
				fmt.Printf("Failed to delete: %d files\n", deleteFailedCount)
			}
		}

		return fmt.Errorf("%d URLs are not accessible", len(failedFiles))
	}

	fmt.Println("\nAll URLs are accessible!")
	return nil
}
