# Provider Implementation Template

This guide shows how to implement a new provider for the Go migration.

## Quick Start

1. Copy the Temurin provider as a template
2. Modify API endpoints and parsing logic
3. Register in main.go
4. Test output

## Provider Template

```go
package vendorname

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"github.com/joschi/java-metadata/internal/models"
)

// Provider implements the Provider interface for [Vendor Name]
type Provider struct {
	client *http.Client
}

// NewProvider creates a new [Vendor Name] provider
func NewProvider() *Provider {
	return &Provider{
		client: &http.Client{},
	}
}

// Name returns the vendor name
func (p *Provider) Name() string {
	return models.VendorXXX // Use constant from models/constants.go
}

// FetchReleases fetches all available releases
func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// 1. Fetch release data from vendor API/website
	//    - GitHub Releases: https://api.github.com/repos/{org}/{repo}/releases
	//    - Web scraping: Use goquery for HTML parsing
	//    - Vendor API: Implement vendor-specific API calls

	// 2. Parse response into vendor-specific structs

	// 3. Convert to []models.Metadata
	var metadata []models.Metadata

	// For each release:
	for _, release := range releases {
		for _, binary := range release.Binaries {
			m := models.Metadata{
				Vendor:       models.VendorXXX,
				Filename:     binary.Filename,
				ReleaseType:  models.ReleaseTypeGA, // or EA
				Version:      binary.Version,
				JavaVersion:  binary.JavaVersion,
				JVMImpl:      binary.JVMImpl, // hotspot, openj9, or graalvm
				OS:           models.NormalizeOS(binary.OS),
				Architecture: models.NormalizeArchitecture(binary.Arch),
				FileType:     extractFileExtension(binary.Filename),
				ImageType:    binary.ImageType, // jdk or jre
				Features:     extractFeatures(binary), // []string
				URL:          binary.DownloadURL,
				// Checksums and size are computed later
				MD5File:      binary.Filename + ".md5",
				SHA1File:     binary.Filename + ".sha1",
				SHA256File:   binary.Filename + ".sha256",
				SHA512File:   binary.Filename + ".sha512",
			}
			metadata = append(metadata, m)
		}
	}

	return metadata, nil
}
```

## Common Patterns

### Pattern 1: GitHub Releases API

Used by: adoptopenjdk, microsoft, sapmachine, semeru*, mandrel, trava*, redhat

```go
func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// Fetch releases from GitHub API
	url := "https://api.github.com/repos/{org}/{repo}/releases?per_page=100"

	// Add GitHub token if available
	req, _ := http.NewRequest("GET", url, nil)
	if token := os.Getenv("GITHUB_API_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := p.client.Do(req)
	// ... parse JSON response

	// GitHub release structure:
	type GitHubRelease struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
}
```

### Pattern 2: Web Scraping

Used by: zulu, liberica, oracle

```go
import "github.com/PuerkitoBio/goquery"

func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// Fetch HTML page
	resp, err := p.client.Get("https://vendor.com/downloads/")
	doc, err := goquery.NewDocumentFromReader(resp.Body)

	// Parse HTML
	doc.Find("a[href$='.tar.gz']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		// Extract metadata from URL/filename
	})
}
```

### Pattern 3: Vendor API

Used by: corretto, graalvm*, oracle-graalvm*, dragonwell*, kona*, openjdk*

```go
func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// Vendor-specific API endpoint
	url := "https://api.vendor.com/v1/releases"
	resp, err := p.client.Get(url)

	// Parse vendor-specific JSON format
	type VendorRelease struct {
		Version  string   `json:"version"`
		Binaries []Binary `json:"binaries"`
	}
}
```

## Common Helper Functions

### Extract File Extension
```go
func extractFileExtension(filename string) string {
	if strings.HasSuffix(filename, ".tar.gz") {
		return "tar.gz"
	}
	ext := filepath.Ext(filename)
	if ext != "" {
		return ext[1:] // Remove leading dot
	}
	return ""
}
```

### Extract Features from Filename
```go
func extractFeatures(filename string) []string {
	var features []string
	if strings.Contains(filename, "fx") || strings.Contains(filename, "javafx") {
		features = append(features, "javafx")
	}
	if strings.Contains(filename, "musl") {
		features = append(features, "musl")
	}
	if strings.Contains(filename, "crac") {
		features = append(features, "crac")
	}
	return features
}
```

### Parse Version from Filename
```go
import "regexp"

func parseVersion(filename string) string {
	// Example: OpenJDK11U-jdk_x64_linux_hotspot_11.0.12_7.tar.gz -> 11.0.12+7
	re := regexp.MustCompile(`(\d+(?:\.\d+)+)(?:[_+](\d+))?`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 2 && matches[2] != "" {
		return matches[1] + "+" + matches[2]
	}
	return matches[1]
}
```

## Registration

Add to `cmd/java-metadata/main.go`:

```go
import (
	"github.com/joschi/java-metadata/internal/providers/vendorname"
)

func runUpdate(...) error {
	registry := providers.NewRegistry()
	registry.Register(temurin.NewProvider())
	registry.Register(vendorname.NewProvider()) // Add your provider
	// ...
}
```

## Testing

### Compare with Bash Output

```bash
# Run Bash version for your vendor
./bin/vendorname.bash docs/metadata/vendor docs/checksums

# Run Go version
./java-metadata update

# Compare outputs
diff -r docs/metadata/vendor/vendorname/all.json <expected-output>
```

### Unit Test Template

```go
package vendorname

import "testing"

func TestFetchReleases(t *testing.T) {
	p := NewProvider()
	releases, err := p.FetchReleases()
	if err != nil {
		t.Fatalf("FetchReleases failed: %v", err)
	}

	if len(releases) == 0 {
		t.Error("Expected at least one release")
	}

	// Validate first release
	r := releases[0]
	if r.Vendor != models.VendorXXX {
		t.Errorf("Expected vendor %s, got %s", models.VendorXXX, r.Vendor)
	}
	if r.Filename == "" {
		t.Error("Filename should not be empty")
	}
	if r.URL == "" {
		t.Error("URL should not be empty")
	}
}
```

## Checklist

Before considering a provider complete:

- [ ] Implements `Provider` interface
- [ ] Returns valid metadata (all required fields)
- [ ] Uses normalization functions (`NormalizeOS`, `NormalizeArchitecture`)
- [ ] Handles pagination if applicable
- [ ] Handles errors gracefully
- [ ] Filters out invalid/unsupported releases
- [ ] Features array is correctly populated
- [ ] Registered in `main.go`
- [ ] Tested against Bash output
- [ ] Unit test added

## Example: Implementing Microsoft Provider

```go
// internal/providers/microsoft/microsoft.go
package microsoft

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"github.com/joschi/java-metadata/internal/models"
)

type Provider struct {
	client *http.Client
}

func NewProvider() *Provider {
	return &Provider{client: &http.Client{}}
}

func (p *Provider) Name() string {
	return models.VendorMicrosoft
}

func (p *Provider) FetchReleases() ([]models.Metadata, error) {
	// Microsoft OpenJDK is on GitHub
	url := "https://api.github.com/repos/microsoft/openjdk/releases?per_page=100"

	req, _ := http.NewRequest("GET", url, nil)
	if token := os.Getenv("GITHUB_API_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	type Release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}

	var releases []Release
	json.Unmarshal(body, &releases)

	var metadata []models.Metadata
	for _, rel := range releases {
		for _, asset := range rel.Assets {
			// Parse filename to extract OS, arch, etc.
			// e.g., microsoft-jdk-11.0.12-linux-x64.tar.gz
			m := parseFilename(asset.Name, asset.URL, rel.TagName)
			if m.Filename != "" {
				metadata = append(metadata, m)
			}
		}
	}

	return metadata, nil
}

func parseFilename(filename, url, version string) models.Metadata {
	// Implementation depends on Microsoft's naming convention
	// Parse OS, arch, image type from filename
	// Return populated Metadata struct
}
```

Register in `main.go`:
```go
import "github.com/joschi/java-metadata/internal/providers/microsoft"

registry.Register(microsoft.NewProvider())
```

## Tips

1. **Start with GitHub-based providers** - easiest to implement
2. **Look at the Bash script** for parsing logic you can port
3. **Use regex sparingly** - parse structured data when possible
4. **Test incrementally** - verify output after each provider
5. **Handle rate limits** - especially for GitHub API (use GITHUB_API_TOKEN)
6. **Skip invalid releases** - return only JDK/JRE with valid metadata
