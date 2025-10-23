# Go Migration Guide

This document describes the Go implementation that replaces the Bash scripts in the `bin` directory.

## Current Status

### ✅ Completed (Phase 1)

1. **Core Infrastructure**
   - Go module initialized: `github.com/joschi/java-metadata`
   - Domain models matching OpenAPI specification (`internal/models/`)
   - Constants and normalization functions (`NormalizeOS`, `NormalizeArchitecture`)
   - Concurrent downloader with checksum computation (MD5, SHA1, SHA256, SHA512)
   - Metadata aggregation logic (creates hierarchical directory structure)

2. **Provider Framework**
   - Provider interface defined (`internal/providers/provider.go`)
   - Provider registry for managing multiple vendors
   - First provider implemented: **Temurin** (Eclipse Adoptium)

3. **CLI Application**
   - Command: `java-metadata update` - fetches and updates metadata
   - Flags: `--metadata-dir`, `--checksum-dir`, `--concurrency`
   - Concurrent provider execution (default: 4 parallel)

## Architecture

```
github.com/joschi/java-metadata/
├── cmd/java-metadata/          # CLI application entry point
│   └── main.go
├── internal/
│   ├── models/                 # Domain models & constants
│   │   ├── metadata.go        # Metadata struct (matches OpenAPI)
│   │   └── constants.go       # Vendors, OS, arch normalization
│   ├── providers/             # Vendor-specific implementations
│   │   ├── provider.go        # Provider interface & registry
│   │   └── temurin/           # Temurin provider
│   │       └── temurin.go
│   ├── downloader/            # HTTP downloader & checksums
│   │   └── downloader.go
│   └── output/                # JSON file writer & aggregation
│       └── metadata.go
└── docs/                      # Generated output (same as Bash)
    ├── metadata/
    │   ├── all.json
    │   ├── vendor/{vendor}/all.json
    │   └── {release_type}/{os}/{arch}/...
    └── checksums/{vendor}/{filename}.{hash}
```

## Building

```bash
# Build the binary
go build -o java-metadata ./cmd/java-metadata

# Run tests
go test ./...

# Run with options
./java-metadata update --metadata-dir=./docs/metadata --checksum-dir=./docs/checksums
```

## Output Compatibility

The Go implementation produces **identical output** to the Bash scripts:

1. **JSON files**: Sorted (equivalent to `jq -S`), pretty-printed with 2-space indentation
2. **Checksum files**: Format compatible with `md5sum`, `sha1sum`, etc.
   ```
   <hex_checksum>  <filename>
   ```
3. **Directory structure**: Matches OpenAPI specification exactly

## Next Steps (Phase 2-5)

### Phase 2: Remaining Providers

Implement the remaining 50 vendor providers:

#### GitHub-based Providers (use GitHub Releases API)
- `adoptopenjdk` (legacy)
- `semeru8`, `semeru11`, `semeru16`, `semeru17`, etc. (IBM Semeru)
- `mandrel` (GraalVM native image)
- `trava8`, `trava11` (Trava OpenJDK)
- `redhat` (Red Hat build of OpenJDK)

#### Web Scraping Providers
- `zulu` (Azul - scrape HTML index)
- `liberica` (BellSoft - API or scraping)
- `sapmachine` (SAP - GitHub Releases)
- `oracle` (Oracle JDK - web scraping)

#### Vendor API Providers
- `corretto` (Amazon - construct URLs)
- `microsoft` (Microsoft OpenJDK - GitHub Releases)
- `graalvm-ce`, `graalvm-community`, `graalvm-legacy` (GraalVM - GitHub)
- `oracle-graalvm`, `oracle-graalvm-ea` (Oracle GraalVM - API)
- `dragonwell8`, `dragonwell11`, `dragonwell17`, `dragonwell21` (Alibaba Dragonwell)
- `kona8`, `kona11`, `kona17`, `kona21` (Tencent Kona)
- `openjdk`, `openjdk-leyden`, `openjdk-loom`, `openjdk-valhalla` (OpenJDK builds)
- `java-se-ri` (Java SE Reference Implementation)
- `ibm` (IBM JDK - legacy)
- `jetbrains` (JetBrains Runtime)
- `bisheng` (Huawei BiSheng JDK)

### Phase 3: Testing & Validation

1. Add unit tests for each provider
2. Integration test comparing Bash vs Go output
3. Implement `validate` command (check URL accessibility)
4. Add GitHub API token support (respect `GITHUB_API_TOKEN` env var)

### Phase 4: Performance Optimization

1. Add connection pooling
2. Implement artifact download caching (skip if already processed)
3. Add progress bars for downloads
4. Optimize concurrent downloads per provider

### Phase 5: Deployment

1. Update `.github/workflows/update.yml` to use Go binary
2. Add Dockerfile for containerized builds
3. Cross-compile binaries for Linux/macOS/Windows
4. Deprecate and remove `bin/*.bash` scripts

## Migration Strategy

### Option 1: Gradual Migration (Recommended)

1. Keep Bash scripts in `bin/`
2. Add Go providers incrementally
3. Run side-by-side comparison tests
4. Switch workflows when all providers implemented

### Option 2: Hybrid Approach

1. Use Go CLI to orchestrate Bash scripts initially
2. Replace Bash scripts one by one with Go providers
3. Transition smoothly without breaking workflows

### Testing Migration

```bash
# Run Bash version
./bin/update.bash

# Run Go version (only Temurin for now)
./java-metadata update

# Compare outputs
diff -r docs/metadata/vendor/temurin/ <expected-output>
```

## Dependencies

Currently using only Go standard library:
- `encoding/json` - JSON marshaling
- `net/http` - HTTP client
- `crypto/*` - Checksum computation
- `sync` - Concurrency primitives

Future dependencies (when needed):
- `github.com/PuerkitoBio/goquery` - HTML parsing for web scraping providers

## Contributing

### Adding a New Provider

1. Create `internal/providers/{vendor}/{vendor}.go`
2. Implement the `Provider` interface:
   ```go
   func (p *Provider) Name() string
   func (p *Provider) FetchReleases() ([]models.Metadata, error)
   ```
3. Register in `cmd/java-metadata/main.go`:
   ```go
   registry.Register(newvendor.NewProvider())
   ```
4. Test output matches Bash version

### Testing Checklist

- [ ] JSON output is sorted and formatted identically
- [ ] All metadata fields match OpenAPI schema
- [ ] Checksum files match `{md5,sha1,sha256,sha512}sum` format
- [ ] Directory structure follows OpenAPI paths
- [ ] Performance is comparable to Bash version

## Known Limitations

1. **Temurin Only**: Currently only Temurin provider is implemented
2. **No Caching**: Downloads artifacts every time (Bash version also does this)
3. **No Resume**: If interrupted, restarts from beginning
4. **Sequential Downloads**: Within a provider, downloads are sequential (could be parallelized)

## Rollback Plan

If issues arise:
1. Revert to Bash scripts (they remain in `bin/`)
2. Identify and fix Go implementation issues
3. Re-test with comparison tests
4. Retry deployment

The Bash scripts should remain in the repository until at least 3 successful production runs of the Go version.
