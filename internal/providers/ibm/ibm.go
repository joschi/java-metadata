package ibm

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
)

// Provider implements the Provider interface for IBM JDK (legacy)
type Provider struct {
	client *http.Client
}

// NewProvider creates a new IBM JDK provider
func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorIBM
}

// FetchReleases fetches all available IBM JDK releases (legacy, Java 7-8 only)
func (p *Provider) FetchReleases(ctx context.Context) ([]models.Metadata, error) {
	var metadata []models.Metadata

	// Fetch index page
	baseURL := "https://public.dhe.ibm.com/ibmdl/export/pub/systems/cloud/runtimes/java/"
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build IBM index request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IBM index: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract version directories (only Java 7-8)
	versionRe := regexp.MustCompile(`<a href="([78]\.[01]\.[0-9]+\.[0-9]+)/">`)
	versionMatches := versionRe.FindAllStringSubmatch(string(body), -1)

	for _, versionMatch := range versionMatches {
		if len(versionMatch) < 2 {
			continue
		}
		version := versionMatch[1]

		// Fetch architectures for this version (Linux only)
		versionURL := fmt.Sprintf("%s%s/linux/", baseURL, version)
		archReq, err := http.NewRequestWithContext(ctx, "GET", versionURL, nil)
		if err != nil {
			logger.Warn("failed to build IBM version request",
				slog.String("version", version),
				slog.Any("error", err),
			)
			continue
		}

		archResp, err := p.client.Do(archReq)
		if err != nil {
			logger.Warn("failed to fetch IBM version",
				slog.String("version", version),
				slog.Any("error", err),
			)
			continue
		}

		archBody, err := io.ReadAll(archResp.Body)
		archResp.Body.Close()
		if err != nil {
			continue
		}

		// Extract architecture directories
		archRe := regexp.MustCompile(`<a href="([a-z0-9_]+)/">`)
		archMatches := archRe.FindAllStringSubmatch(string(archBody), -1)

		for _, archMatch := range archMatches {
			if len(archMatch) < 2 {
				continue
			}
			arch := archMatch[1]

			// Fetch files for this architecture
			archURL := fmt.Sprintf("%s%s/linux/%s/", baseURL, version, arch)
			fileReq, err := http.NewRequestWithContext(ctx, "GET", archURL, nil)
			if err != nil {
				logger.Warn("failed to build IBM arch request",
					slog.String("version", version),
					slog.String("arch", arch),
					slog.Any("error", err),
				)
				continue
			}

			fileResp, err := p.client.Do(fileReq)
			if err != nil {
				logger.Warn("failed to fetch IBM arch",
					slog.String("version", version),
					slog.String("arch", arch),
					slog.Any("error", err),
				)
				continue
			}

			fileBody, err := io.ReadAll(fileResp.Body)
			fileResp.Body.Close()
			if err != nil {
				continue
			}

			// Extract .tgz files
			fileRe := regexp.MustCompile(`<a href="(.*\.tgz)">`)
			fileMatches := fileRe.FindAllStringSubmatch(string(fileBody), -1)

			for _, fileMatch := range fileMatches {
				if len(fileMatch) < 2 {
					continue
				}
				filename := fileMatch[1]

				// Skip sfj (Small Footprint JRE) files
				if strings.Contains(filename, "sfj") {
					continue
				}

				url := fmt.Sprintf("%s%s/linux/%s/%s", baseURL, version, arch, filename)
				m := p.parseFilename(filename, url, version, arch)
				if m.Filename != "" {
					metadata = append(metadata, m)
				}
			}
		}
	}

	return metadata, nil
}

// parseFilename parses IBM JDK filename
func (p *Provider) parseFilename(filename, url, version, arch string) models.Metadata {
	// Determine image type
	imageType := models.ImageTypeJRE
	if strings.Contains(filename, "sdk") {
		imageType = models.ImageTypeJDK
	}

	// Extract file extension
	ext := "tgz"
	if strings.HasSuffix(filename, ".tar.gz") {
		ext = "tar.gz"
	}

	return models.Metadata{
		Vendor:       models.VendorIBM,
		Filename:     filename,
		ReleaseType:  models.ReleaseTypeGA,
		Version:      version,
		JavaVersion:  version,
		JVMImpl:      models.JVMImplOpenJ9,
		OS:           models.OSLinux,
		Architecture: models.NormalizeArchitecture(arch),
		FileType:     ext,
		ImageType:    imageType,
		Features:     []string{},
		URL:          url,
		MD5File:      filename + ".md5",
		SHA1File:     filename + ".sha1",
		SHA256File:   filename + ".sha256",
		SHA512File:   filename + ".sha512",
	}
}
