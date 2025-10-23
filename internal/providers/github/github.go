package github

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

// Release represents a GitHub release
type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		ContentType string `json:"content_type"`
	} `json:"assets"`
}

// FilenameParser is a function that parses metadata from a filename
type FilenameParser func(filename, url, tagName string) models.Metadata

// GenericProvider is a generic GitHub Releases provider
type GenericProvider struct {
	client     *http.Client
	vendorName string
	org        string
	repo       string
	parser     FilenameParser
}

// NewGenericProvider creates a new generic GitHub provider
func NewGenericProvider(vendorName, org, repo string, parser FilenameParser) *GenericProvider {
	return &GenericProvider{
		client:     &http.Client{},
		vendorName: vendorName,
		org:        org,
		repo:       repo,
		parser:     parser,
	}
}

// Name returns the vendor name
func (p *GenericProvider) Name() string {
	return p.vendorName
}

// FetchReleases fetches all releases from GitHub
func (p *GenericProvider) FetchReleases() ([]models.Metadata, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=100", p.org, p.repo)

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

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}

	var metadata []models.Metadata
	for _, release := range releases {
		for _, asset := range release.Assets {
			// Skip non-application content types
			if !strings.HasPrefix(asset.ContentType, "application") && asset.ContentType != "" {
				continue
			}

			// Parse the filename
			m := p.parser(asset.Name, asset.DownloadURL, release.TagName)
			if m.Filename != "" {
				metadata = append(metadata, m)
			}
		}
	}

	return metadata, nil
}

// ParseSemeruFilename parses IBM Semeru filename
func ParseSemeruFilename(vendorName string) FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		// Skip symbol files
		if strings.Contains(filename, "symbols") || strings.Contains(filename, "-debuginfo-") {
			return models.Metadata{}
		}

		// Parse version from tag: jdk-11.0.12+7_openj9-0.27.0
		versionRe := regexp.MustCompile(`^jdk-(.*)_openj9-(.*)$`)
		versionMatches := versionRe.FindStringSubmatch(tagName)
		if len(versionMatches) < 3 {
			return models.Metadata{}
		}
		javaVersion := versionMatches[1]
		openj9Version := versionMatches[2]
		version := javaVersion + "_openj9-" + openj9Version

		// Parse filename
		// Format: ibm-semeru-open-{jre|jdk}_{arch}_{os}_{version}_openj9-{openj9version}.{ext}
		// Or RPM: ibm-semeru-open-{version}-(jre|jdk)-{rpmversion}.{arch}.rpm

		var imageType, os, arch, ext string

		if strings.HasSuffix(filename, ".rpm") {
			rpmRe := regexp.MustCompile(`^ibm-semeru-open-[0-9]+-(jre|jdk)-.+\.(x86_64|s390x|ppc64|ppc64le|aarch64)\.rpm$`)
			matches := rpmRe.FindStringSubmatch(filename)
			if len(matches) < 3 {
				return models.Metadata{}
			}
			imageType = matches[1]
			arch = matches[2]
			os = "linux"
			ext = "rpm"
		} else {
			stdRe := regexp.MustCompile(`^ibm-semeru-open-(jre|jdk)_(x64|x86-32|s390x|ppc64|ppc64le|aarch64)_(aix|linux|mac|windows)_.+_openj9-.+\.(tar\.gz|zip|msi)$`)
			matches := stdRe.FindStringSubmatch(filename)
			if len(matches) < 5 {
				return models.Metadata{}
			}
			imageType = matches[1]
			arch = matches[2]
			os = matches[3]
			ext = matches[4]
		}

		return models.Metadata{
			Vendor:       vendorName,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      version,
			JavaVersion:  javaVersion,
			JVMImpl:      models.JVMImplOpenJ9,
			OS:           models.NormalizeOS(os),
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
}

// ParseGraalVMFilename parses GraalVM filename
func ParseGraalVMFilename(vendorName string) FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		if !strings.Contains(filename, "java") || strings.Contains(filename, "polyglot") {
			return models.Metadata{}
		}
		var os, arch, ext string
		if strings.Contains(filename, "linux") {
			os = "linux"
		} else if strings.Contains(filename, "darwin") || strings.Contains(filename, "macos") {
			os = "macosx"
		} else if strings.Contains(filename, "windows") {
			os = "windows"
		}
		if strings.Contains(filename, "amd64") || strings.Contains(filename, "x64") {
			arch = "x64"
		} else if strings.Contains(filename, "aarch64") {
			arch = "aarch64"
		}
		if strings.HasSuffix(filename, ".tar.gz") {
			ext = "tar.gz"
		} else if strings.HasSuffix(filename, ".zip") {
			ext = "zip"
		}
		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}
		return models.Metadata{
			Vendor:       vendorName,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      tagName,
			JavaVersion:  tagName,
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
}

// ParseMandrelFilename parses Mandrel filename
func ParseMandrelFilename() FilenameParser {
	return ParseGraalVMFilename(models.VendorMandrel)
}

// ParseDragonwellFilename parses Dragonwell filename
func ParseDragonwellFilename(vendorName string) FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		if strings.Contains(filename, "_src") || strings.HasSuffix(filename, ".txt") {
			return models.Metadata{}
		}
		var os, arch, ext string
		if strings.Contains(filename, "_linux_") {
			os = "linux"
		} else if strings.Contains(filename, "_windows_") {
			os = "windows"
		}
		if strings.Contains(filename, "_x64") {
			arch = "x64"
		} else if strings.Contains(filename, "_aarch64") {
			arch = "aarch64"
		}
		if strings.HasSuffix(filename, ".tar.gz") {
			ext = "tar.gz"
		} else if strings.HasSuffix(filename, ".zip") {
			ext = "zip"
		}
		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}
		imageType := models.ImageTypeJDK
		if strings.Contains(filename, "_jre_") {
			imageType = models.ImageTypeJRE
		}
		return models.Metadata{
			Vendor:       vendorName,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      tagName,
			JavaVersion:  tagName,
			JVMImpl:      models.JVMImplHotSpot,
			OS:           models.NormalizeOS(os),
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
}

// ParseTravaFilename parses Trava OpenJDK filename
func ParseTravaFilename(vendorName string) FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		if strings.HasSuffix(filename, ".sha256") || strings.HasSuffix(filename, ".sig") {
			return models.Metadata{}
		}
		var os, arch, ext string
		if strings.Contains(filename, "-linux-") {
			os = "linux"
		} else if strings.Contains(filename, "-windows-") {
			os = "windows"
		}
		if strings.Contains(filename, "-x64-") || strings.Contains(filename, "-x86_64-") {
			arch = "x64"
		}
		if strings.HasSuffix(filename, ".tar.gz") {
			ext = "tar.gz"
		} else if strings.HasSuffix(filename, ".zip") {
			ext = "zip"
		}
		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}
		return models.Metadata{
			Vendor:       vendorName,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      tagName,
			JavaVersion:  tagName,
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
}

// ParseJetBrainsFilename parses JetBrains Runtime filename
func ParseJetBrainsFilename() FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		if !strings.HasPrefix(filename, "jbr") {
			return models.Metadata{}
		}
		var os, arch, ext string
		if strings.Contains(filename, "-linux-") {
			os = "linux"
		} else if strings.Contains(filename, "-osx-") {
			os = "macosx"
		} else if strings.Contains(filename, "-windows-") {
			os = "windows"
		}
		if strings.Contains(filename, "-x64-") || strings.Contains(filename, "_x64.") {
			arch = "x64"
		} else if strings.Contains(filename, "-aarch64-") {
			arch = "aarch64"
		}
		if strings.HasSuffix(filename, ".tar.gz") {
			ext = "tar.gz"
		} else if strings.HasSuffix(filename, ".zip") {
			ext = "zip"
		}
		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}
		return models.Metadata{
			Vendor:       models.VendorJetBrains,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      tagName,
			JavaVersion:  tagName,
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
}

// ParseKonaFilename parses Tencent Kona filename
func ParseKonaFilename(vendorName string) FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		// Skip checksums and signatures
		if strings.HasSuffix(filename, ".sha256") || strings.HasSuffix(filename, ".sig") {
			return models.Metadata{}
		}

		var os, arch, ext string
		var features []string

		// Pattern: TencentKona-{version}_jdk_{fiber}_{os}-{arch}.{ext}
		if strings.HasPrefix(filename, "TencentKona-") {
			if strings.Contains(filename, "_linux-") || strings.Contains(filename, "_linux_") {
				os = "linux"
			} else if strings.Contains(filename, "_macosx-") || strings.Contains(filename, "_macosx_") {
				os = "macosx"
			} else if strings.Contains(filename, "_windows-") || strings.Contains(filename, "_windows_") {
				os = "windows"
			}

			if strings.Contains(filename, "-aarch64") || strings.Contains(filename, "_aarch64") {
				arch = "aarch64"
			} else if strings.Contains(filename, "-x86_64") || strings.Contains(filename, "_x86_64") || strings.Contains(filename, "-x64") {
				arch = "x64"
			}

			if strings.Contains(filename, "_fiber_") {
				features = append(features, "fiber")
			}

			if strings.HasSuffix(filename, ".tar.gz") {
				ext = "tar.gz"
			} else if strings.HasSuffix(filename, ".zip") {
				ext = "zip"
			}
		} else if strings.HasSuffix(filename, ".tgz") {
			// Pattern: TencentKona{version}.tgz (old format)
			os = "linux"
			arch = "x64"
			ext = "tgz"
		}

		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}

		return models.Metadata{
			Vendor:       vendorName,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      tagName,
			JavaVersion:  tagName,
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
}

// ParseOpenJDKFilename parses OpenJDK filename (generic for openjdk, leyden, loom, valhalla)
func ParseOpenJDKFilename(vendorName string) FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		// Skip checksums
		if strings.HasSuffix(filename, ".sha256") || strings.HasSuffix(filename, ".txt") {
			return models.Metadata{}
		}

		var os, arch, ext string
		
		if strings.Contains(filename, "_linux_") || strings.Contains(filename, "_linux-") {
			os = "linux"
		} else if strings.Contains(filename, "_macos_") || strings.Contains(filename, "_osx_") {
			os = "macosx"
		} else if strings.Contains(filename, "_windows_") {
			os = "windows"
		}

		if strings.Contains(filename, "_x64_") || strings.Contains(filename, "-x64_") {
			arch = "x64"
		} else if strings.Contains(filename, "_aarch64_") {
			arch = "aarch64"
		}

		if strings.HasSuffix(filename, ".tar.gz") {
			ext = "tar.gz"
		} else if strings.HasSuffix(filename, ".zip") {
			ext = "zip"
		}

		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}

		return models.Metadata{
			Vendor:       vendorName,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeEA, // Most OpenJDK builds are EA
			Version:      tagName,
			JavaVersion:  tagName,
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
}

// ParseRedHatFilename parses Red Hat OpenJDK filename
func ParseRedHatFilename() FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		if strings.HasSuffix(filename, ".sha256") {
			return models.Metadata{}
		}

		var os, arch, ext string
		
		if strings.Contains(filename, "_linux_") {
			os = "linux"
		} else if strings.Contains(filename, "_windows_") {
			os = "windows"
		}

		if strings.Contains(filename, "_x64") || strings.Contains(filename, "-x64") {
			arch = "x64"
		}

		if strings.HasSuffix(filename, ".tar.gz") {
			ext = "tar.gz"
		} else if strings.HasSuffix(filename, ".zip") {
			ext = "zip"
		} else if strings.HasSuffix(filename, ".msi") {
			ext = "msi"
		}

		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}

		return models.Metadata{
			Vendor:       models.VendorRedHat,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      tagName,
			JavaVersion:  tagName,
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
}

// ParseBiShengFilename parses BiSheng JDK filename
func ParseBiShengFilename() FilenameParser {
	return func(filename, url, tagName string) models.Metadata {
		if strings.HasSuffix(filename, ".sha256") {
			return models.Metadata{}
		}

		var os, arch, ext string
		
		if strings.Contains(filename, "-linux-") {
			os = "linux"
		}

		if strings.Contains(filename, "-x86_64") || strings.Contains(filename, "-x64") {
			arch = "x64"
		} else if strings.Contains(filename, "-aarch64") {
			arch = "aarch64"
		}

		if strings.HasSuffix(filename, ".tar.gz") {
			ext = "tar.gz"
		}

		if os == "" || arch == "" || ext == "" {
			return models.Metadata{}
		}

		return models.Metadata{
			Vendor:       models.VendorBisheng,
			Filename:     filename,
			ReleaseType:  models.ReleaseTypeGA,
			Version:      tagName,
			JavaVersion:  tagName,
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
}
