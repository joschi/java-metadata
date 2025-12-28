package javase

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

// Provider implements the Provider interface for Java SE Reference Implementation
type Provider struct {
	client *http.Client
}

// NewProvider creates a new Java SE RI provider
func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorJavaSERI
}

// FetchReleases fetches all available Java SE Reference Implementation releases
func (p *Provider) FetchReleases(ctx context.Context) ([]models.Metadata, error) {
	versions := []string{"7", "8-MR3", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25"}

	var allURLs []string

	// Fetch download pages for each version
	for _, ver := range versions {
		url := fmt.Sprintf("https://jdk.java.net/java-se-ri/%s", ver)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			logger.Warn(ctx, "failed to build Java SE RI request",
				slog.String("provider", p.Name()),
				slog.String("version", ver),
				slog.Any("error", err),
			)
			continue
		}

		resp, err := p.client.Do(req)
		if err != nil {
			logger.Warn(ctx, "failed to fetch Java SE RI download page",
				slog.String("provider", p.Name()),
				slog.String("version", ver),
				slog.Any("error", err),
			)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// Extract download URLs
		re := regexp.MustCompile(`href="(https://download\.java\.net/.*/openjdk-[^"]*\.(tar\.gz|zip))"`)
		matches := re.FindAllStringSubmatch(string(body), -1)

		for _, match := range matches {
			if len(match) > 1 {
				// Skip source files
				if !strings.Contains(match[1], "_src") {
					allURLs = append(allURLs, match[1])
				}
			}
		}
	}

	var metadata []models.Metadata
	seen := make(map[string]bool)

	for _, url := range allURLs {
		// Extract filename from URL
		parts := strings.Split(url, "/")
		filename := parts[len(parts)-1]

		if seen[filename] {
			continue
		}
		seen[filename] = true

		m := p.parseFilename(filename, url)
		if m.Filename != "" {
			metadata = append(metadata, m)
		}
	}

	return metadata, nil
}

// parseFilename parses Java SE RI filename
// Pattern: openjdk-{version}_{os}-{arch}.{ext}
func (p *Provider) parseFilename(filename, url string) models.Metadata {
	re := regexp.MustCompile(`^openjdk-([0-9ub-]+[^_]*)[-_](linux|osx|windows)-(aarch64|x64-musl|x64|i586).*\.(tar\.gz|zip)$`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) < 5 {
		return models.Metadata{}
	}

	version := matches[1]
	os := matches[2]
	arch := matches[3]
	ext := matches[4]

	// Determine release type
	releaseType := models.ReleaseTypeGA
	if strings.Contains(version, "-ea") {
		releaseType = models.ReleaseTypeEA
	}

	// Handle musl
	var features []string
	if strings.Contains(arch, "musl") {
		features = append(features, "musl")
		arch = strings.Replace(arch, "-musl", "", 1)
	}

	return models.Metadata{
		Vendor:       models.VendorJavaSERI,
		Filename:     filename,
		ReleaseType:  releaseType,
		Version:      version,
		JavaVersion:  version,
		JVMImpl:      models.JVMImplHotSpot,
		OS:           models.NormalizeOS(os),
		Architecture: models.NormalizeArchitecture(arch),
		FileType:     ext,
		ImageType:    models.ImageTypeJDK,
		Features:     features,
		URL:          url,
		MD5File:      filename + ".md5",
		SHA1File:     filename + ".sha1",
		SHA256File:   filename + ".sha256",
		SHA512File:   filename + ".sha512",
	}
}
