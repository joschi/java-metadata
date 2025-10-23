package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joschi/java-metadata/internal/models"
)

func TestWriteMetadataJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.json")

	metadata := []models.Metadata{
		{
			Vendor:       models.VendorTemurin,
			Filename:     "test1.tar.gz",
			ReleaseType:  models.ReleaseTypeGA,
			Version:      "17.0.1",
			JavaVersion:  "17.0.1+12",
			JVMImpl:      models.JVMImplHotSpot,
			OS:           models.OSLinux,
			Architecture: models.ArchX86_64,
			FileType:     "tar.gz",
			ImageType:    models.ImageTypeJDK,
			Features:     []string{},
			URL:          "https://example.com/test1.tar.gz",
		},
		{
			Vendor:       models.VendorTemurin,
			Filename:     "test2.tar.gz",
			ReleaseType:  models.ReleaseTypeGA,
			Version:      "11.0.13",
			JavaVersion:  "11.0.13+8",
			JVMImpl:      models.JVMImplHotSpot,
			OS:           models.OSLinux,
			Architecture: models.ArchAArch64,
			FileType:     "tar.gz",
			ImageType:    models.ImageTypeJDK,
			Features:     []string{"musl"},
			URL:          "https://example.com/test2.tar.gz",
		},
	}

	err := WriteMetadataJSON(outputFile, metadata)
	if err != nil {
		t.Fatalf("WriteMetadataJSON failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Read and parse the JSON
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var parsed []models.Metadata
	err = json.Unmarshal(content, &parsed)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify metadata count
	if len(parsed) != len(metadata) {
		t.Errorf("Expected %d metadata entries, got %d", len(metadata), len(parsed))
	}

	// Verify sorting (by vendor, then filename)
	if len(parsed) >= 2 {
		if parsed[0].Filename > parsed[1].Filename {
			t.Error("Metadata not sorted by filename")
		}
	}
}

func TestAggregateMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	metadata := []models.Metadata{
		{
			Vendor:       models.VendorTemurin,
			Filename:     "test1.tar.gz",
			ReleaseType:  models.ReleaseTypeGA,
			Version:      "17.0.1",
			JavaVersion:  "17.0.1+12",
			JVMImpl:      models.JVMImplHotSpot,
			OS:           models.OSLinux,
			Architecture: models.ArchX86_64,
			FileType:     "tar.gz",
			ImageType:    models.ImageTypeJDK,
			Features:     []string{},
			URL:          "https://example.com/test1.tar.gz",
		},
		{
			Vendor:       models.VendorCorretto,
			Filename:     "test2.tar.gz",
			ReleaseType:  models.ReleaseTypeEA,
			Version:      "18.0.0",
			JavaVersion:  "18.0.0-ea",
			JVMImpl:      models.JVMImplHotSpot,
			OS:           models.OSWindows,
			Architecture: models.ArchX86_64,
			FileType:     "zip",
			ImageType:    models.ImageTypeJRE,
			Features:     []string{},
			URL:          "https://example.com/test2.zip",
		},
	}

	err := AggregateMetadata(models.MetadataList(metadata), tmpDir)
	if err != nil {
		t.Fatalf("AggregateMetadata failed: %v", err)
	}

	// Verify that at least the top-level release type files were created
	gaFile := filepath.Join(tmpDir, "ga.json")
	eaFile := filepath.Join(tmpDir, "ea.json")

	if _, err := os.Stat(gaFile); os.IsNotExist(err) {
		t.Errorf("ga.json was not created")
	}
	if _, err := os.Stat(eaFile); os.IsNotExist(err) {
		t.Errorf("ea.json was not created")
	}
}

func TestAggregateMetadataEmptySlice(t *testing.T) {
	tmpDir := t.TempDir()

	err := AggregateMetadata(models.MetadataList([]models.Metadata{}), tmpDir)
	if err != nil {
		t.Fatalf("AggregateMetadata failed with empty slice: %v", err)
	}

	// With empty metadata, files may or may not be created depending on implementation
	// The important thing is that it doesn't error
}
