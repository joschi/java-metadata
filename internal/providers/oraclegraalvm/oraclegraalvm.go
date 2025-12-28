package oraclegraalvm

import (
	"context"
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

// Provider implements the Provider interface for Oracle GraalVM
type Provider struct {
	client      *http.Client
	vendorName  string
	releaseType string
}

// NewProvider creates a new Oracle GraalVM provider (GA releases)
func NewProvider() *Provider {
	return &Provider{
		client:      &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		vendorName:  models.VendorOracleGraalVM,
		releaseType: models.ReleaseTypeGA,
	}
}

// NewEAProvider creates a new Oracle GraalVM provider for Early Access releases
func NewEAProvider() *Provider {
	return &Provider{
		client:      &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
		vendorName:  models.VendorOracleGraalVM,
		releaseType: models.ReleaseTypeEA,
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	if p.releaseType == models.ReleaseTypeEA {
		return p.vendorName + "-ea"
	}
	return p.vendorName
}

// FetchReleases fetches all available Oracle GraalVM releases
func (p *Provider) FetchReleases(ctx context.Context) ([]models.Metadata, error) {
	var metadata []models.Metadata

	// Fetch current releases from main downloads page
	currentMetadata := p.fetchCurrentReleases(ctx)
	metadata = append(metadata, currentMetadata...)

	// Only fetch archives for GA releases
	if p.releaseType == models.ReleaseTypeGA {
		// Fetch archive downloads for major versions
		for _, version := range []string{"17", "19", "20", "21", "22", "23", "24"} {
			archiveMetadata := p.fetchArchiveMetadata(ctx, version)
			metadata = append(metadata, archiveMetadata...)
		}
	}

	return metadata, nil
}

// fetchCurrentReleases fetches current releases from Oracle downloads page
func (p *Provider) fetchCurrentReleases(ctx context.Context) []models.Metadata {
	var metadata []models.Metadata

	url := "https://www.oracle.com/java/technologies/downloads/"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Warn(ctx, "failed to build Oracle GraalVM current request", slog.Any("error", err))
		return metadata
	}

	resp, err := p.client.Do(req)
	if err != nil {
		logger.Warn(ctx, "failed to fetch Oracle GraalVM current releases", slog.Any("error", err))
		return metadata
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return metadata
	}

	// Extract current versions
	versionRe := regexp.MustCompile(`<h3 id="graalvmjava([0-9]{2})">GraalVM for JDK (.+) downloads</h3>`)
	matches := versionRe.FindAllStringSubmatch(string(body), -1)

	for _, match := range matches {
		if len(match) > 2 {
			majorVersion := match[1]
			version := match[2]

			// Generate download URLs for all platforms
			platforms := []struct {
				os   string
				arch string
				ext  string
			}{
				{"linux", "aarch64", "tar.gz"},
				{"linux", "x64", "tar.gz"},
				{"macos", "aarch64", "tar.gz"},
				{"macos", "x64", "tar.gz"},
				{"windows", "x64", "zip"},
			}

			for _, platform := range platforms {
				filename := fmt.Sprintf("graalvm-jdk-%s_%s-%s_bin.%s", version, platform.os, platform.arch, platform.ext)
				url := fmt.Sprintf("https://download.oracle.com/graalvm/%s/archive/%s", majorVersion, filename)

				m := p.parseFilename(filename, url)
				if m.Filename != "" {
					metadata = append(metadata, m)
				}
			}
		}
	}

	return metadata
}

// fetchArchiveMetadata fetches metadata from Oracle GraalVM archive pages
func (p *Provider) fetchArchiveMetadata(ctx context.Context, majorVersion string) []models.Metadata {
	var metadata []models.Metadata

	url := fmt.Sprintf("https://www.oracle.com/java/technologies/javase/graalvm-jdk%s-archive-downloads.html", majorVersion)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Warn(ctx, "failed to build Oracle GraalVM archive request",
			slog.String("version", majorVersion),
			slog.Any("error", err),
		)
		return metadata
	}

	resp, err := p.client.Do(req)
	if err != nil {
		logger.Warn(ctx, "failed to fetch Oracle GraalVM archive",
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
	re := regexp.MustCompile(`<a href="(https://download\.oracle\.com/graalvm/.+/archive/(graalvm-jdk-.+_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm)))">`)
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

// parseFilename parses Oracle GraalVM filename
// Pattern: graalvm-jdk-{version}_{os}-{arch}_bin.{ext}
func (p *Provider) parseFilename(filename, url string) models.Metadata {
	re := regexp.MustCompile(`^graalvm-jdk-([0-9+.]{2,})_(linux|macos|windows)-(x64|aarch64)_bin\.(tar\.gz|zip|msi|dmg|exe|deb|rpm)$`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) < 5 {
		return models.Metadata{}
	}

	version := matches[1]
	os := matches[2]
	arch := matches[3]
	ext := matches[4]

	// Determine release type based on version string
	releaseType := p.releaseType
	if strings.Contains(version, "-ea") {
		releaseType = models.ReleaseTypeEA
	}

	return models.Metadata{
		Vendor:       p.vendorName,
		Filename:     filename,
		ReleaseType:  releaseType,
		Version:      version,
		JavaVersion:  version,
		JVMImpl:      models.JVMImplGraalVM,
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
