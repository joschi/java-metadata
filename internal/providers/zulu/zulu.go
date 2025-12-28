package zulu

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/joschi/java-metadata/internal/logger"
	"github.com/joschi/java-metadata/internal/models"
)

// Provider implements the Provider interface for Azul Zulu
type Provider struct {
	client *http.Client
}

// NewProvider creates a new Zulu provider
func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorZulu
}

// FetchReleases fetches all available releases for Azul Zulu
func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// Fetch the index page
	url := "https://static.azul.com/zulu/bin/"
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index: %w", err)
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
	// Pattern: <a href=".../(zulu...-(linux|macosx|win|solaris)_(arch).(tar.gz|zip|msi|dmg))">
	linkRe := regexp.MustCompile(`<a href="[^"]*/(zulu.+?-(linux|macosx|win|solaris)_(musl_x64|musl_aarch64|x64|i686|aarch32hf|aarch32sf|aarch64|ppc64|sparcv9)\.(tar\.gz|zip|msi|dmg))">`)
	matches := linkRe.FindAllStringSubmatch(string(body), -1)

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

		// Skip zre files (not jre/jdk)
		if strings.Contains(filename, "zre") {
			continue
		}

		// Parse metadata from filename
		m := p.parseFilename(filename)
		if m.Filename != "" {
			metadata = append(metadata, m)
		}
	}

	return metadata, nil
}

// parseFilename parses metadata from Zulu filename
// Format: zulu{version}-{release_type}-{image_type}{java_version}-{os}_{arch}.{ext}
func (p *Provider) parseFilename(filename string) models.Metadata {
	// Regex matching Zulu naming pattern
	re := regexp.MustCompile(`^zulu([0-9+_.]{2,})-(?:(ca-crac|ca-fx-dbg|ca-fx|ca-hl|ca-dbg|ea-cp3|ca|ea|dbg|oem|beta)-)?(jdk|jre)(.*)-(linux|macosx|win|solaris)_(musl_aarch64|musl_x64|x64|i686|aarch32hf|aarch32sf|aarch64|ppc64|sparcv9)\.(.+)$`)
	matches := re.FindStringSubmatch(filename)

	if len(matches) < 8 {
		logger.Warn("failed to parse Zulu filename", slog.String("filename", filename))
		return models.Metadata{}
	}

	version := matches[1]
	releaseTypeStr := matches[2]
	imageType := matches[3]
	javaVersion := matches[4]
	os := matches[5]
	arch := matches[6]
	ext := matches[7]

	// Normalize release type
	releaseType := normalizeReleaseType(releaseTypeStr)
	if releaseType == "" {
		// Skip debug builds and other unsupported types
		return models.Metadata{}
	}

	// Parse features
	features := parseFeatures(releaseTypeStr, arch)

	// Normalize architecture (handle musl variants)
	normalizedArch := arch
	switch arch {
	case "musl_aarch64":
		normalizedArch = "aarch64"
	case "musl_x64":
		normalizedArch = "x64"
	}

	return models.Metadata{
		Vendor:       models.VendorZulu,
		Filename:     filename,
		ReleaseType:  releaseType,
		Version:      version,
		JavaVersion:  javaVersion,
		JVMImpl:      models.JVMImplHotSpot,
		OS:           models.NormalizeOS(os),
		Architecture: models.NormalizeArchitecture(normalizedArch),
		FileType:     ext,
		ImageType:    imageType,
		Features:     features,
		URL:          "https://static.azul.com/zulu/bin/" + filename,
		MD5File:      filename + ".md5",
		SHA1File:     filename + ".sha1",
		SHA256File:   filename + ".sha256",
		SHA512File:   filename + ".sha512",
	}
}

// normalizeReleaseType converts Zulu release type strings to standard format
func normalizeReleaseType(releaseType string) string {
	switch releaseType {
	case "ca", "ca-fx", "ca-crac", "":
		return models.ReleaseTypeGA
	case "ea", "beta":
		return models.ReleaseTypeEA
	case "ca-dbg", "ca-fx-dbg", "dbg":
		// Skip debug builds
		return ""
	default:
		return ""
	}
}

// parseFeatures extracts features from release type and architecture
func parseFeatures(releaseType, arch string) []string {
	var features []string

	if releaseType == "ca-fx" || releaseType == "ca-fx-dbg" {
		features = append(features, "javafx")
	}

	if releaseType == "ca-crac" {
		features = append(features, "crac")
	}

	if arch == "musl_x64" || arch == "musl_aarch64" {
		features = append(features, "musl")
	}

	return features
}
