package corretto

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/joschi/java-metadata/internal/downloader"
	"github.com/joschi/java-metadata/internal/logger"
	"github.com/joschi/java-metadata/internal/models"
)

// Provider implements the Provider interface for Amazon Corretto
type Provider struct {
	client     *http.Client
	downloader *downloader.Downloader
}

// NewProvider creates a new Corretto provider
func NewProvider() *Provider {
	return &Provider{
		client:     &http.Client{},
		downloader: downloader.NewDownloader(),
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorCorretto
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// FetchReleases fetches all available releases for Amazon Corretto
func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// Corretto has multiple GitHub repos by version
	repos := []string{
		"corretto-8", "corretto-11", "corretto-17", "corretto-18", "corretto-19",
		"corretto-20", "corretto-21", "corretto-22", "corretto-23", "corretto-24",
		"corretto-25", "corretto-jdk",
	}

	// Fetch all versions from GitHub releases
	var allVersions []string
	for _, repo := range repos {
		versions, err := p.fetchVersionsFromRepo(repo)
		if err != nil {
			logger.Warn("failed to fetch from repository",
				slog.String("repo", repo),
				slog.Any("error", err),
			)
			continue
		}
		allVersions = append(allVersions, versions...)
	}

	// Construct URLs and check which ones exist
	var metadata []models.Metadata

	for _, version := range allVersions {
		// Try all combinations of OS, arch, extension, image type
		for _, osConfig := range getOSConfigs() {
			for _, arch := range osConfig.Archs {
				for _, ext := range osConfig.Extensions {
					for _, imageType := range getImageTypesForOSAndExt(osConfig.OS, ext) {
						filename := constructFilename(version, osConfig.OS, arch, ext, imageType)
						url := fmt.Sprintf("https://corretto.aws/downloads/resources/%s/%s", version, filename)

						// Check if URL exists (HEAD request)
						if err := p.downloader.CheckURLExists(url); err == nil {
							// URL exists, create metadata
							m := models.Metadata{
								Vendor:       models.VendorCorretto,
								Filename:     filename,
								ReleaseType:  models.ReleaseTypeGA,
								Version:      version,
								JavaVersion:  version,
								JVMImpl:      models.JVMImplHotSpot,
								OS:           models.NormalizeOS(osConfig.OS),
								Architecture: models.NormalizeArchitecture(arch),
								FileType:     ext,
								ImageType:    imageType,
								Features:     getFeatures(osConfig.OS),
								URL:          url,
								MD5File:      filename + ".md5",
								SHA1File:     filename + ".sha1",
								SHA256File:   filename + ".sha256",
								SHA512File:   filename + ".sha512",
							}
							metadata = append(metadata, m)
						}
					}
				}
			}
		}
	}

	return metadata, nil
}

// fetchVersionsFromRepo fetches version tags from a Corretto GitHub repository
func (p *Provider) fetchVersionsFromRepo(repo string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/corretto/%s/releases?per_page=100", repo)

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

	var releases []GitHubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}

	var versions []string
	for _, release := range releases {
		versions = append(versions, release.TagName)
	}

	return versions, nil
}

// OSConfig represents OS configuration with architectures and extensions
type OSConfig struct {
	OS         string
	Archs      []string
	Extensions []string
}

// getOSConfigs returns all OS configurations
func getOSConfigs() []OSConfig {
	return []OSConfig{
		{
			OS:         "linux",
			Archs:      []string{"x86", "x64", "aarch64", "armv7"},
			Extensions: []string{"tar.gz", "rpm", "deb"},
		},
		{
			OS:         "alpine-linux",
			Archs:      []string{"x64", "aarch64"},
			Extensions: []string{"tar.gz"},
		},
		{
			OS:         "macosx",
			Archs:      []string{"x64", "aarch64"},
			Extensions: []string{"tar.gz", "pkg"},
		},
		{
			OS:         "windows",
			Archs:      []string{"x64", "x86"},
			Extensions: []string{"zip", "msi"},
		},
	}
}

// getImageTypesForOSAndExt returns image types for a given OS and extension
func getImageTypesForOSAndExt(os, ext string) []string {
	if os == "windows" && ext == "zip" {
		return []string{models.ImageTypeJRE, models.ImageTypeJDK}
	}
	return []string{models.ImageTypeJDK}
}

// constructFilename constructs a Corretto download filename
func constructFilename(version, os, arch, ext, imageType string) string {
	if imageType == models.ImageTypeJDK && os != "windows" {
		// Most platforms don't include image type in filename for JDK
		return fmt.Sprintf("amazon-corretto-%s-%s-%s.%s", version, os, arch, ext)
	}
	return fmt.Sprintf("amazon-corretto-%s-%s-%s-%s.%s", version, os, arch, imageType, ext)
}

// getFeatures returns features based on OS
func getFeatures(os string) []string {
	if os == "alpine-linux" {
		return []string{"musl"}
	}
	return []string{}
}
