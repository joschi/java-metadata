package microsoft

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/joschi/java-metadata/internal/logger"
	"github.com/joschi/java-metadata/internal/models"
)

// Provider implements the Provider interface for Microsoft OpenJDK
type Provider struct {
	client *http.Client
}

// NewProvider creates a new Microsoft provider
func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorMicrosoft
}

// FetchReleases fetches all available releases for Microsoft OpenJDK
func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// Fetch the download page
	url := "https://docs.microsoft.com/en-us/java/openjdk/download"
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch download page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse download links from HTML
	// Pattern: <a href="https://aka.ms/download-jdk/microsoft-jdk-{version}-{os}-{arch}.{ext}">
	re := regexp.MustCompile(`<a href="https://aka\.ms/download-jdk/(microsoft-jdk-.+?-(linux|macos|macOS|windows)-(x64|aarch64)\.(tar\.gz|zip|msi|dmg|pkg))"`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var metadata []models.Metadata
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		filename := match[1]

		// Skip duplicates
		if seen[filename] {
			continue
		}
		seen[filename] = true

		// Parse metadata from filename
		m := p.parseFilename(filename)
		if m.Filename != "" {
			metadata = append(metadata, m)
		}
	}

	return metadata, nil
}

// parseFilename parses metadata from Microsoft JDK filename
// Format: microsoft-jdk-{version}-{os}-{arch}.{ext}
func (p *Provider) parseFilename(filename string) models.Metadata {
	// Regex: microsoft-jdk-{version}-{os}-{arch}.{ext}
	re := regexp.MustCompile(`^microsoft-jdk-([0-9+.]{3,})-(linux|macos|macOS|windows)-(x64|aarch64)\.(.+)$`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) < 5 {
		logger.Warn("failed to parse Microsoft JDK filename", slog.String("filename", filename))
		return models.Metadata{}
	}

	version := matches[1]
	os := matches[2]
	arch := matches[3]
	ext := matches[4]

	// Determine release type (aarch64 builds are EA)
	releaseType := models.ReleaseTypeGA
	if arch == "aarch64" {
		releaseType = models.ReleaseTypeEA
	}

	return models.Metadata{
		Vendor:       models.VendorMicrosoft,
		Filename:     filename,
		ReleaseType:  releaseType,
		Version:      version,
		JavaVersion:  version,
		JVMImpl:      models.JVMImplHotSpot,
		OS:           models.NormalizeOS(os),
		Architecture: models.NormalizeArchitecture(arch),
		FileType:     ext,
		ImageType:    models.ImageTypeJDK,
		Features:     []string{},
		URL:          "https://aka.ms/download-jdk/" + filename,
		MD5File:      filename + ".md5",
		SHA1File:     filename + ".sha1",
		SHA256File:   filename + ".sha256",
		SHA512File:   filename + ".sha512",
	}
}
