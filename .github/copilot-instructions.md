# Copilot Instructions for `java-metadata`

Use these project-specific guidelines to work productively in this repo. Focus on the Go implementation (current), with Bash in `bin/` as legacy reference.

## Big Picture
- Goal: Collect JRE/JDK release metadata across vendors, publish JSON under `docs/metadata/` and checksum files under `docs/checksums/`.
- CLI: `java-metadata update` fetches provider releases, downloads artifacts to compute checksums/sizes, writes JSON, and aggregates hierarchy; `java-metadata validate` HEAD-checks URLs and optionally deletes failing files.
- Why this structure: Providers focus on discovery and normalization; checksum/size is computed centrally for consistency and performance.

## Architecture Map
- Entry point: `cmd/java-metadata/main.go` — commands (`update`, `validate`), flags (`--metadata-dir`, `--checksum-dir`, `--concurrency`, `--download-concurrency`, `--no-progress`, `--max-retries`, logging flags), concurrency orchestration, aggregation.
- Providers: `internal/providers/provider.go` (interface) + `internal/providers/allproviders/` (registry) with patterns:
  - GitHub Releases (majority via `internal/providers/github/` generic provider)
  - Web scraping (regex/goquery) for Zulu, Oracle, etc.
  - Vendor APIs (Temurin, AdoptOpenJDK, Corretto)
- Models: `internal/models/metadata.go` (OpenAPI-aligned schema) + `constants.go` (vendors, OS/arch enums, normalization helpers `NormalizeOS`, `NormalizeArchitecture`).
- Output: `internal/output/metadata.go` — sorted JSON writers (`WriteMetadataJSON`, `WriteSingleMetadataJSON`) and hierarchical aggregation `{release_type}/{os}/{arch}/{image_type}/{jvm_impl}/{vendor}.json`.
- Downloads: `internal/downloader/downloader.go` — optimized HTTP client, progress bars, retry/backoff, single-pass MD5/SHA1/SHA256/SHA512, checksum file writers, `CheckURLExists` for validate.
- Logging: `internal/logger/logger.go` — structured `log/slog` levels with global flags.

## Conventions (Do This Here)
- Providers return metadata WITHOUT checksums/sizes; these are computed in the update pipeline after downloads.
- Normalize OS/arch via `internal/models.NormalizeOS()` and `NormalizeArchitecture()`; output must use canonical names.
- Sort output consistently: metadata is sorted by vendor+filename and features arrays are alphabetized; JSON is pretty-printed and stable.
- Use `GITHUB_TOKEN` (preferred) or `GITHUB_API_TOKEN` for GitHub API; attach as `Authorization: token <value>`.

## Workflows & Commands
- Build: `go build -o java-metadata ./cmd/java-metadata`
- Test: `go test ./...` (fast, covers models, downloader, output, providers, registry)
- Update: `./java-metadata update --metadata-dir=./docs/metadata --checksum-dir=./docs/checksums --concurrency=4 --download-concurrency=3`
- Validate: `./java-metadata validate --metadata-dir=./docs/metadata --concurrency=10 --delete`
- Legacy full update: `./bin/update.bash` (GNU parallel, aggregates into `docs/metadata/all.json`), and URL check: `./bin/validate.bash <metadata-file>`.

## Provider Patterns & Examples
- Register all vendors in `internal/providers/allproviders/allproviders.go` (e.g., generic GitHub providers for Kona/OpenJDK/Dragonwell/JetBrains/RedHat/BiSheng/Liberica).
- Provider contract:
  ```go
  type Provider interface {
    Name() string
    FetchReleases() ([]models.Metadata, error)
  }
  ```
- Typical GitHub provider: build releases URL, include token if present, parse assets into `models.Metadata`, normalize OS/arch, derive `file_type`, `image_type`, features, `URL`, and populate `MD5File`/`SHA*File` names (checksums filled later).

## Output Structure (Expectations)
- Combined: `docs/metadata/all.json` and per-vendor `docs/metadata/vendor/<vendor>/all.json` plus individual `<filename>.json` files.
- Aggregation writes filtered JSON at each path level: `{release_type}`, `{os}`, `{arch}`, `{image_type}`, `{jvm_impl}`, `{vendor}`.
- Checksum files: `docs/checksums/<vendor>/<filename>.{md5,sha1,sha256,sha512}` formatted as `<hex>  <filename>`.

## CI & Tokens
- Workflows run Go commands (`go run ./cmd/java-metadata ...`), not Bash; keep Bash scripts for historical parity.
- To avoid GitHub rate limits (39 providers use GitHub), set `GITHUB_TOKEN` in env; fallback `GITHUB_API_TOKEN` retained for Bash compatibility.

## When Adding/Editing Providers
- Reuse existing parsers in `internal/providers/github/` where possible; prefer framework over bespoke logic.
- Keep parsing robust to file naming variations (version suffixes, musl, javafx, crac, openj9/graalvm impl tags).
- Verify provider output against expected shape: fields present, normalized enums, valid `image_type` and `jvm_impl`.
- Register new provider in `allproviders.RegisterAll()`.

## References
- Data model: `docs/openapi.yaml` (schema)
- High-level docs: `CLAUDE.md`, `GO_IMPLEMENTATION_SUMMARY.md`, `GO_MIGRATION.md`
- Legacy helpers/patterns: `bin/functions.bash`
