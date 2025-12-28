package temurin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joschi/java-metadata/internal/logger"
	"github.com/joschi/java-metadata/internal/models"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Provider implements the Provider interface for Eclipse Temurin (Adoptium)
type Provider struct {
	client     *http.Client
	vendorName string
	apiVendor  string
}

// NewProvider creates a new Temurin provider
func NewProvider() *Provider {
	return NewProviderWithVendor(models.VendorTemurin, "adoptium")
}

// NewProviderWithVendor creates a provider with custom vendor name and API parameter
// This allows reusing the Adoptium API for different vendors (e.g., adoptopenjdk)
func NewProviderWithVendor(vendorName, apiVendor string) *Provider {
	return &Provider{
		client:     &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		vendorName: vendorName,
		apiVendor:  apiVendor,
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return p.vendorName
}

// AdoptiumAvailableReleases represents the response from /v3/info/available_releases
type AdoptiumAvailableReleases struct {
	AvailableReleases []int `json:"available_releases"`
}

// AdoptiumRelease represents a release from the Adoptium API
type AdoptiumRelease struct {
	ReleaseType string              `json:"release_type"`
	VersionData AdoptiumVersionData `json:"version_data"`
	Binaries    []AdoptiumBinary    `json:"binaries"`
}

// AdoptiumVersionData contains version information
type AdoptiumVersionData struct {
	OpenJDKVersion string `json:"openjdk_version"`
	Semver         string `json:"semver"`
}

// AdoptiumBinary represents a binary download
type AdoptiumBinary struct {
	Architecture string          `json:"architecture"`
	OS           string          `json:"os"`
	HeapSize     string          `json:"heap_size"`
	ImageType    string          `json:"image_type"`
	JVMImpl      string          `json:"jvm_impl"`
	Package      AdoptiumPackage `json:"package"`
}

// AdoptiumPackage contains download information
type AdoptiumPackage struct {
	Link string `json:"link"`
	Name string `json:"name"`
}

// FetchReleases fetches all available releases for Temurin
func (p *Provider) FetchReleases(ctx context.Context) ([]models.Metadata, error) {
	// Fetch available release versions
	releases, err := p.fetchAvailableReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch available releases: %w", err)
	}

	var allMetadata []models.Metadata

	// Fetch release data for each version
	for _, release := range releases {
		metadata, err := p.fetchReleaseMetadata(ctx, release)
		if err != nil {
			logger.Warn(ctx, "failed to fetch release",
				slog.String("provider", p.Name()),
				slog.Int("release", release),
				slog.Any("error", err),
			)
			continue
		}
		allMetadata = append(allMetadata, metadata...)
	}

	return allMetadata, nil
}

// fetchAvailableReleases fetches the list of available release versions
func (p *Provider) fetchAvailableReleases(ctx context.Context) ([]int, error) {
	url := "https://api.adoptium.net/v3/info/available_releases"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases AdoptiumAvailableReleases
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}

	return releases.AvailableReleases, nil
}

// fetchReleaseMetadata fetches metadata for a specific release version
func (p *Provider) fetchReleaseMetadata(ctx context.Context, release int) ([]models.Metadata, error) {
	var allReleases []AdoptiumRelease

	// Paginate through releases
	page := 0
	for {
		url := fmt.Sprintf("https://api.adoptium.net/v3/assets/feature_releases/%d/ga?page=%d&page_size=20&project=jdk&sort_order=ASC&vendor=%s", release, page, p.apiVendor)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, err
		}

		// If no content, we've reached the end
		if resp.StatusCode != http.StatusOK || len(body) == 0 {
			break
		}

		var pageReleases []AdoptiumRelease
		if err := json.Unmarshal(body, &pageReleases); err != nil {
			return nil, err
		}

		// If empty page, we're done
		if len(pageReleases) == 0 {
			break
		}

		allReleases = append(allReleases, pageReleases...)
		page++
	}

	// Convert to metadata
	var metadata []models.Metadata
	for _, release := range allReleases {
		for _, binary := range release.Binaries {
			// Skip non-JRE, non-JDK images
			if binary.ImageType != models.ImageTypeJRE && binary.ImageType != models.ImageTypeJDK {
				continue
			}

			m := p.convertToMetadata(release, binary)
			metadata = append(metadata, m)
		}
	}

	return metadata, nil
}

// convertToMetadata converts Adoptium API data to our metadata format
func (p *Provider) convertToMetadata(release AdoptiumRelease, binary AdoptiumBinary) models.Metadata {
	filename := binary.Package.Name
	ext := extractFileExtension(filename)
	version := normalizeVersion(binary.JVMImpl, filename, release.VersionData.Semver)
	features := normalizeFeatures(binary.HeapSize, binary.OS)

	return models.Metadata{
		Vendor:       p.vendorName,
		Filename:     filename,
		ReleaseType:  models.ReleaseTypeGA, // Temurin only provides GA releases
		Version:      version,
		JavaVersion:  release.VersionData.OpenJDKVersion,
		JVMImpl:      binary.JVMImpl,
		OS:           models.NormalizeOS(binary.OS),
		Architecture: models.NormalizeArchitecture(binary.Architecture),
		FileType:     ext,
		ImageType:    binary.ImageType,
		Features:     features,
		URL:          binary.Package.Link,
		MD5:          "", // Will be computed during download
		MD5File:      filename + ".md5",
		SHA1:         "", // Will be computed during download
		SHA1File:     filename + ".sha1",
		SHA256:       "", // Will be computed during download
		SHA256File:   filename + ".sha256",
		SHA512:       "", // Will be computed during download
		SHA512File:   filename + ".sha512",
		Size:         0, // Will be computed during download
	}
}

// extractFileExtension extracts the file extension from a filename
func extractFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == ".gz" && strings.HasSuffix(filename, ".tar.gz") {
		return "tar.gz"
	}
	if ext != "" {
		return ext[1:] // Remove leading dot
	}
	return ""
}

// normalizeVersion normalizes the version string, adding OpenJ9 version if applicable
func normalizeVersion(jvmImpl, filename, version string) string {
	if jvmImpl == models.JVMImplOpenJ9 && strings.Contains(filename, "openj9") && !strings.Contains(version, "openj9") {
		// Extract openj9 version from filename
		re := regexp.MustCompile(`[_-](openj9[-_]\d+\.\d+\.\d+[a-z]?)`)
		matches := re.FindStringSubmatch(filename)
		if len(matches) > 1 {
			return version + "." + matches[1]
		}
	}
	return version
}

// normalizeFeatures returns the feature list based on heap size and OS
func normalizeFeatures(heapSize, os string) []string {
	var features []string
	if heapSize == "large" {
		features = append(features, "large_heap")
	}
	if os == "alpine-linux" {
		features = append(features, "musl")
	}
	return features
}
