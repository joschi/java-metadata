package sapmachine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/joschi/java-metadata/internal/models"
)

// Provider implements the Provider interface for SAPMachine
type Provider struct {
	client *http.Client
}

// NewProvider creates a new SAPMachine provider
func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorSAPMachine
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		ContentType string `json:"content_type"`
	} `json:"assets"`
}

// FetchReleases fetches all available releases for SAPMachine
func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	url := "https://api.github.com/repos/SAP/SapMachine/releases?per_page=100"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add GitHub token if available
	if token := os.Getenv("GITHUB_API_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []GitHubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}

	var metadata []models.Metadata

	for _, release := range releases {
		for _, asset := range release.Assets {
			// Skip non-application content types
			if !strings.HasPrefix(asset.ContentType, "application") {
				continue
			}

			// Skip symbol files
			if strings.Contains(asset.Name, "symbols") {
				continue
			}

			m := p.parseFilename(asset.Name, asset.DownloadURL, release.TagName)
			if m.Filename != "" {
				metadata = append(metadata, m)
			}
		}
	}

	return metadata, nil
}

// parseFilename parses metadata from SAPMachine filename
// Format 1: sapmachine-{image_type}-{version}_{os}-{arch}-{features}_bin.{ext}
// Format 2: sapmachine-{image_type}-{version}.{arch}.rpm
func (p *Provider) parseFilename(filename, url, tagName string) models.Metadata {
	var m models.Metadata

	// Try RPM format first
	rpmRe := regexp.MustCompile(`^sapmachine-(jdk|jre)-([0-9].+)\.(aarch64|ppc64le|x86_64)\.rpm$`)
	if matches := rpmRe.FindStringSubmatch(filename); len(matches) == 4 {
		m = models.Metadata{
			Vendor:       models.VendorSAPMachine,
			Filename:     filename,
			ReleaseType:  getReleaseType(matches[2]),
			Version:      matches[2],
			JavaVersion:  matches[2],
			JVMImpl:      models.JVMImplHotSpot,
			OS:           models.OSLinux,
			Architecture: models.NormalizeArchitecture(matches[3]),
			FileType:     "rpm",
			ImageType:    matches[1],
			Features:     []string{},
			URL:          url,
			MD5File:      filename + ".md5",
			SHA1File:     filename + ".sha1",
			SHA256File:   filename + ".sha256",
			SHA512File:   filename + ".sha512",
		}
		return m
	}

	// Try standard format
	standardRe := regexp.MustCompile(`^sapmachine-(jdk|jre)-([0-9].+)_(aix|linux|macos|osx|windows)-(x64|aarch64|ppc64|ppc64le)-?(.*)_bin\.(.+)$`)
	if matches := standardRe.FindStringSubmatch(filename); len(matches) == 7 {
		features := parseFeatures(matches[5])

		m = models.Metadata{
			Vendor:       models.VendorSAPMachine,
			Filename:     filename,
			ReleaseType:  getReleaseType(matches[2]),
			Version:      matches[2],
			JavaVersion:  matches[2],
			JVMImpl:      models.JVMImplHotSpot,
			OS:           models.NormalizeOS(matches[3]),
			Architecture: models.NormalizeArchitecture(matches[4]),
			FileType:     matches[6],
			ImageType:    matches[1],
			Features:     features,
			URL:          url,
			MD5File:      filename + ".md5",
			SHA1File:     filename + ".sha1",
			SHA256File:   filename + ".sha256",
			SHA512File:   filename + ".sha512",
		}
		return m
	}

	// If parsing failed, return empty metadata
	fmt.Printf("Failed to parse SAPMachine filename: %s\n", filename)
	return models.Metadata{}
}

// getReleaseType determines release type from version string
func getReleaseType(version string) string {
	if strings.Contains(version, "ea") {
		return models.ReleaseTypeEA
	}
	return models.ReleaseTypeGA
}

// parseFeatures parses features from the features string
func parseFeatures(featuresStr string) []string {
	if featuresStr == "" {
		return []string{}
	}

	// Split by common separators and filter empty strings
	parts := strings.FieldsFunc(featuresStr, func(r rune) bool {
		return r == '-' || r == '_'
	})

	var features []string
	for _, part := range parts {
		if part != "" {
			features = append(features, part)
		}
	}

	return features
}
