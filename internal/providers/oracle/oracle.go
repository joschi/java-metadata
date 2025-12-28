package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/joschi/java-metadata/internal/logger"
	"github.com/joschi/java-metadata/internal/models"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Provider implements the Provider interface for Oracle JDK
type Provider struct {
	client     *http.Client
	vendorName string
}

// NewProvider creates a new Oracle JDK provider
func NewProvider() *Provider {
	return &Provider{
		client:     &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		vendorName: models.VendorOracle,
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return p.vendorName
}

// FetchReleases fetches all available Oracle JDK releases
func (p *Provider) FetchReleases(ctx context.Context) ([]models.Metadata, error) {
	var metadata []models.Metadata

	// Fetch latest versions from Oracle API
	latestReq, err := http.NewRequestWithContext(ctx, "GET", "https://java.oraclecloud.com/javaVersions", nil)
	if err != nil {
		logger.Warn(ctx, "failed to build Oracle latest request", slog.Any("error", err))
	}

	latestResp, err := p.client.Do(latestReq)
	if err != nil {
		logger.Warn(ctx, "failed to fetch Oracle latest versions", slog.Any("error", err))
	} else {
		body, err := io.ReadAll(latestResp.Body)
		latestResp.Body.Close()
		if err == nil {
			var versions struct {
				Items []struct {
					LatestReleaseVersion string `json:"latestReleaseVersion"`
				} `json:"items"`
			}
			if err := json.Unmarshal(body, &versions); err == nil {
				for _, item := range versions.Items {
					releaseMetadata := p.fetchVersionMetadata(ctx, item.LatestReleaseVersion)
					metadata = append(metadata, releaseMetadata...)
				}
			}
		}
	}

	// Fetch archive downloads for major versions
	for _, version := range []string{"17", "18", "19", "20", "21", "22", "23", "24"} {
		archiveMetadata := p.fetchArchiveMetadata(ctx, version)
		metadata = append(metadata, archiveMetadata...)
	}

	return metadata, nil
}

// fetchVersionMetadata fetches metadata for a specific version from Oracle API
func (p *Provider) fetchVersionMetadata(ctx context.Context, version string) []models.Metadata {
	var metadata []models.Metadata

	url := fmt.Sprintf("https://java.oraclecloud.com/javaReleases/%s", version)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Warn(ctx, "failed to build Oracle version request",
			slog.String("version", version),
			slog.Any("error", err),
		)
		return metadata
	}

	resp, err := p.client.Do(req)
	if err != nil {
		logger.Warn(ctx, "failed to fetch Oracle version",
			slog.String("version", version),
			slog.Any("error", err),
		)
		return metadata
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return metadata
	}

	var release struct {
		LicenseDetails struct {
			LicenseType string `json:"licenseType"`
		} `json:"licenseDetails"`
		Artifacts []struct {
			DownloadURL string `json:"downloadUrl"`
		} `json:"artifacts"`
	}

	if err := json.Unmarshal(body, &release); err != nil {
		return metadata
	}

	// Skip OTN licensed versions
	if release.LicenseDetails.LicenseType == "OTN" {
		logger.Info(ctx, "skipping OTN licensed version", slog.String("version", version))
		return metadata
	}

	for _, artifact := range release.Artifacts {
		m := p.parseOracleURL(artifact.DownloadURL)
		if m.Filename != "" {
			metadata = append(metadata, m)
		}
	}

	return metadata
}

// fetchArchiveMetadata fetches metadata from Oracle archive pages
func (p *Provider) fetchArchiveMetadata(ctx context.Context, majorVersion string) []models.Metadata {
	var metadata []models.Metadata

	url := fmt.Sprintf("https://www.oracle.com/java/technologies/javase/jdk%s-archive-downloads.html", majorVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Warn(ctx, "failed to build Oracle archive request",
			slog.String("version", majorVersion),
			slog.Any("error", err),
		)
		return metadata
	}

	resp, err := p.client.Do(req)
	if err != nil {
		logger.Warn(ctx, "failed to fetch Oracle archive",
			slog.String("version", majorVersion),
			slog.Any("error", err),
		)
		return metadata
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return metadata
	}

	// Extract download links from HTML
	re := regexp.MustCompile(`<a href="(https://download\.oracle\.com/java/.+/archive/(jdk-.+_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm)))">`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			url := match[1]
			filename := match[2]

			if seen[filename] {
				continue
			}
			seen[filename] = true

			m := p.parseFilename(filename, url)
			if m.Filename != "" {
				metadata = append(metadata, m)
			}
		}
	}

	return metadata
}

// parseOracleURL parses Oracle download URL
func (p *Provider) parseOracleURL(url string) models.Metadata {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return models.Metadata{}
	}
	filename := parts[len(parts)-1]
	return p.parseFilename(filename, url)
}

// parseFilename parses Oracle JDK filename
// Pattern: jdk-{version}_{os}-{arch}_bin.{ext}
func (p *Provider) parseFilename(filename, url string) models.Metadata {
	re := regexp.MustCompile(`^jdk-([0-9+.]{2,})_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm)$`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) < 5 {
		return models.Metadata{}
	}

	version := matches[1]
	os := matches[2]
	arch := matches[3]
	ext := matches[4]

	return models.Metadata{
		Vendor:       p.vendorName,
		Filename:     filename,
		ReleaseType:  models.ReleaseTypeGA,
		Version:      version,
		JavaVersion:  version,
		JVMImpl:      models.JVMImplHotSpot,
		OS:           models.NormalizeOS(os),
		Architecture: models.NormalizeArchitecture(arch),
		FileType:     ext,
		ImageType:    models.ImageTypeJDK,
		Features:     []string{},
		URL:          url,
		MD5File:      filename + ".md5",
		SHA1File:     filename + ".sha1",
		SHA256File:   filename + ".sha256",
		SHA512File:   filename + ".sha512",
	}
}
