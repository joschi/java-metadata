package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/joschi/java-metadata/internal/models"
)

// WriteMetadataJSON writes metadata to a JSON file with sorted, pretty-printed output
func WriteMetadataJSON(outputPath string, metadata models.MetadataList) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Sort metadata for consistent output (equivalent to jq -S)
	sortMetadata(metadata)

	// Marshal with indentation
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// WriteSingleMetadataJSON writes a single metadata entry to a JSON file
func WriteSingleMetadataJSON(outputPath string, metadata models.Metadata) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// sortMetadata sorts a metadata list for consistent output
func sortMetadata(metadata models.MetadataList) {
	sort.Slice(metadata, func(i, j int) bool {
		// Sort by vendor, then by filename
		if metadata[i].Vendor != metadata[j].Vendor {
			return metadata[i].Vendor < metadata[j].Vendor
		}
		return metadata[i].Filename < metadata[j].Filename
	})

	// Sort features arrays for each entry
	for i := range metadata {
		if metadata[i].Features != nil {
			sort.Strings(metadata[i].Features)
		}
	}
}

// AggregateMetadata creates the hierarchical directory structure and filtered JSON files
// This replicates the aggregate_metadata function from functions.bash
func AggregateMetadata(allMetadata models.MetadataList, metadataDir string) error {
	// Sort the metadata first
	sortMetadata(allMetadata)

	// Extract unique values
	releaseTypes := extractUniqueValues(allMetadata, func(m models.Metadata) string { return m.ReleaseType })
	operatingSystems := extractUniqueValues(allMetadata, func(m models.Metadata) string { return m.OS })
	architectures := extractUniqueValues(allMetadata, func(m models.Metadata) string { return m.Architecture })
	imageTypes := []string{models.ImageTypeJRE, models.ImageTypeJDK}
	jvmImpls := []string{models.JVMImplHotSpot, models.JVMImplOpenJ9, models.JVMImplGraalVM}
	vendors := extractUniqueValues(allMetadata, func(m models.Metadata) string { return m.Vendor })

	// Create hierarchical structure: {release_type}/{os}/{arch}/{image_type}/{jvm_impl}/{vendor}.json
	for _, releaseType := range releaseTypes {
		releaseTypeDir := filepath.Join(metadataDir, releaseType)
		releaseTypeMetadata := filterMetadata(allMetadata, func(m models.Metadata) bool {
			return m.ReleaseType == releaseType
		})
		if err := WriteMetadataJSON(filepath.Join(metadataDir, releaseType+".json"), releaseTypeMetadata); err != nil {
			return err
		}

		for _, os := range operatingSystems {
			osDir := filepath.Join(releaseTypeDir, os)
			osMetadata := filterMetadata(releaseTypeMetadata, func(m models.Metadata) bool {
				return m.OS == os
			})
			if len(osMetadata) == 0 {
				continue
			}
			if err := WriteMetadataJSON(filepath.Join(releaseTypeDir, os+".json"), osMetadata); err != nil {
				return err
			}

			for _, arch := range architectures {
				archDir := filepath.Join(osDir, arch)
				archMetadata := filterMetadata(osMetadata, func(m models.Metadata) bool {
					return m.Architecture == arch
				})
				if len(archMetadata) == 0 {
					continue
				}
				if err := WriteMetadataJSON(filepath.Join(osDir, arch+".json"), archMetadata); err != nil {
					return err
				}

				for _, imageType := range imageTypes {
					imageTypeDir := filepath.Join(archDir, imageType)
					imageTypeMetadata := filterMetadata(archMetadata, func(m models.Metadata) bool {
						return m.ImageType == imageType
					})
					if len(imageTypeMetadata) == 0 {
						continue
					}
					if err := WriteMetadataJSON(filepath.Join(archDir, imageType+".json"), imageTypeMetadata); err != nil {
						return err
					}

					for _, jvmImpl := range jvmImpls {
						jvmImplDir := filepath.Join(imageTypeDir, jvmImpl)
						jvmImplMetadata := filterMetadata(imageTypeMetadata, func(m models.Metadata) bool {
							return m.JVMImpl == jvmImpl
						})
						if len(jvmImplMetadata) == 0 {
							continue
						}
						if err := WriteMetadataJSON(filepath.Join(imageTypeDir, jvmImpl+".json"), jvmImplMetadata); err != nil {
							return err
						}

						for _, vendor := range vendors {
							vendorMetadata := filterMetadata(jvmImplMetadata, func(m models.Metadata) bool {
								return m.Vendor == vendor
							})
							if len(vendorMetadata) == 0 {
								continue
							}
							if err := WriteMetadataJSON(filepath.Join(jvmImplDir, vendor+".json"), vendorMetadata); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// extractUniqueValues extracts unique values from metadata using the provided selector function
func extractUniqueValues(metadata models.MetadataList, selector func(models.Metadata) string) []string {
	valueMap := make(map[string]bool)
	for _, m := range metadata {
		valueMap[selector(m)] = true
	}

	values := make([]string, 0, len(valueMap))
	for v := range valueMap {
		values = append(values, v)
	}
	sort.Strings(values)
	return values
}

// filterMetadata filters metadata based on the provided predicate
func filterMetadata(metadata models.MetadataList, predicate func(models.Metadata) bool) models.MetadataList {
	result := make(models.MetadataList, 0)
	for _, m := range metadata {
		if predicate(m) {
			result = append(result, m)
		}
	}
	return result
}
