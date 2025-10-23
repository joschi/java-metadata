# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository collects and publishes metadata about available JRE/JDK distributions from multiple vendors (Temurin, Zulu, Corretto, GraalVM, etc.). The project is being migrated from Bash scripts to Go. Both implementations fetch release information, compute checksums, and generate JSON metadata files that are published to GitHub Pages at `https://joschi.github.io/java-metadata/`.

**Migration Status**: Go implementation in progress (Phase 1 complete - core infrastructure and Temurin provider working). See `GO_MIGRATION.md` for details.

## Architecture

### Go Implementation (New)

The Go implementation is located in:
- **`cmd/java-metadata/`**: CLI application (`go build -o java-metadata ./cmd/java-metadata`)
- **`internal/models/`**: Domain models matching OpenAPI spec, normalization functions
- **`internal/providers/`**: Vendor-specific provider implementations (interface + registry pattern)
- **`internal/downloader/`**: HTTP downloader with concurrent checksums
- **`internal/output/`**: JSON writer and hierarchical aggregation

Key command:
```bash
./java-metadata update --metadata-dir=./docs/metadata --checksum-dir=./docs/checksums
```

### Bash Implementation (Legacy)

The Bash scripts in `bin/` are being phased out but remain functional:

#### Script Organization

- **`bin/functions.bash`**: Shared utility functions used by all vendor scripts
  - `download_github_releases()`: Fetches release data from GitHub API (respects `GITHUB_API_TOKEN`)
  - `hash_file()`: Computes md5/sha1/sha256/sha512 checksums
  - `metadata_json()`: Generates standardized JSON metadata using `jo`
  - `normalize_os()` / `normalize_arch()`: Standardizes OS and architecture names
  - `aggregate_metadata()`: Creates hierarchical metadata structure by release type, OS, arch, image type, and JVM impl

- **`bin/<vendor>.bash`**: Individual vendor-specific scripts (51 total)
  - Each script fetches releases for a specific vendor (e.g., `temurin.bash`, `zulu.bash`, `corretto.bash`)
  - Scripts follow a common pattern: fetch release list, parse metadata, download/compute checksums, output JSON
  - Some vendors have multiple scripts for different major versions (e.g., `dragonwell8.bash`, `dragonwell11.bash`)

- **`bin/update.bash`**: Main orchestration script
  - Runs all vendor scripts in parallel using GNU `parallel` (4 concurrent jobs)
  - Aggregates individual vendor JSON files into `docs/metadata/all.json`
  - Calls `aggregate_metadata()` to create hierarchical directory structure under `docs/metadata/`

- **`bin/validate.bash`**: Validation script
  - Checks if URLs in metadata are still accessible via HTTP HEAD requests
  - Used by the validation workflow to remove stale metadata

### Output Structure

- **`docs/metadata/`**: JSON metadata files
  - `all.json`: Combined metadata for all vendors and releases
  - `vendor/<vendor>/all.json`: Per-vendor metadata
  - Hierarchical structure: `{release_type}/{os}/{arch}/{image_type}/{jvm_impl}/{vendor}.json`

- **`docs/checksums/`**: Checksum files compatible with `md5sum`, `sha1sum`, etc.
  - Organized by vendor: `{vendor}/{filename}.{md5,sha1,sha256,sha512}`

## Common Development Commands

### Go Implementation

```bash
# Build the Go binary
go build -o java-metadata ./cmd/java-metadata

# Run tests
go test ./...

# Update metadata (currently only Temurin provider implemented)
./java-metadata update

# With custom directories
./java-metadata update --metadata-dir=./docs/metadata --checksum-dir=./docs/checksums --concurrency=4
```

### Bash Implementation (Legacy)

```bash
# Run ShellCheck on all bash scripts
shellcheck -x bin/*.bash

# Install required dependencies (Ubuntu/Debian)
sudo apt -y install jq jo perl curl parallel

# Run the full update (fetches metadata for all vendors)
./bin/update.bash

# Run a single vendor script
./bin/temurin.bash docs/metadata/vendor docs/checksums

# Validate a single metadata file (checks if URL is accessible)
./bin/validate.bash docs/metadata/vendor/temurin/some-release.json
```

## GitHub Actions Workflows

- **`test.yml`**: Runs ShellCheck on all bash scripts (triggered on push/PR to main)
- **`update.yml`**: Runs `bin/update.bash` every 6 hours and commits changes (also triggered on push to main)
- **`validate.yml`**: Runs `bin/validate.bash` daily to remove invalid metadata entries

## Adding a New Vendor

### Go Implementation (Preferred)

1. Create `internal/providers/<vendor>/<vendor>.go`
2. Implement the `Provider` interface:
   ```go
   func (p *Provider) Name() string { return models.Vendor<Name> }
   func (p *Provider) FetchReleases() ([]models.Metadata, error)
   ```
3. Register in `cmd/java-metadata/main.go`: `registry.Register(vendor.NewProvider())`
4. Return metadata without checksums/sizes (computed during download phase)
5. Test output matches expected format

### Bash Implementation (Legacy)

1. Create `bin/<vendor>.bash` following the pattern of existing vendor scripts
2. Source `bin/functions.bash` for shared utilities
3. Use `metadata_json()` to generate standardized JSON output
4. Add the vendor script to the `vendors` array in `bin/update.bash`
5. Test the script independently before adding to the orchestration

## Key Conventions

- All scripts use strict bash mode: `set -e`, `set -Euo pipefail`
- Temporary directories are cleaned up with `trap 'rm -rf ${TEMP_DIR}' EXIT`
- OS names are normalized to: `linux`, `macosx`, `windows`, `solaris`, `aix`
- Architecture names are normalized to: `x86_64`, `i686`, `aarch64`, `arm32`, `ppc64le`, etc.
- Release types: `ga` (general availability/stable) or `ea` (early access)
- Image types: `jdk` or `jre`
- JVM implementations: `hotspot`, `openj9`, `graalvm`
