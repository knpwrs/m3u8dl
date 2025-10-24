# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`m3u8dl` is a Go-based CLI utility that downloads M3U8 playlists and all referenced files recursively. It mirrors M3U8 playlists by downloading all segments, nested playlists, encryption keys, subtitles, and other resources, optionally rewriting URLs to create a fully local copy that can be played without network access.

M3U8 files are playlists used for HTTP Live Streaming (HLS), containing URLs to video/audio segments and other resources.

## Development Commands

### Building
```bash
go build -o m3u8dl .
```

### Running
```bash
# Run directly with go
go run . https://example.com/playlist.m3u8

# After building
./m3u8dl https://example.com/playlist.m3u8

# Common usage examples
./m3u8dl -o ./downloads https://example.com/playlist.m3u8
./m3u8dl --flatten -v https://example.com/playlist.m3u8
./m3u8dl --include .m3u8,.ts https://example.com/playlist.m3u8
```

### Testing
```bash
go test ./...                 # Run all tests
go test -v ./...             # Verbose output
go test -run TestName ./...  # Run specific test
go test -cover ./...         # With coverage
```

### Linting
```bash
go vet ./...                 # Go's built-in linter
golangci-lint run           # If golangci-lint is installed
```

### Dependencies
```bash
go mod tidy                  # Clean up dependencies
go mod download             # Download dependencies
go get <package>            # Add new dependency
```

## Architecture

### Project Structure

```
m3u8dl/
├── main.go                          # CLI entry point
├── cmd/
│   └── root.go                      # Cobra root command and flags
├── internal/
│   ├── downloader/
│   │   ├── downloader.go           # Main download orchestration with worker pools
│   │   ├── m3u8.go                 # M3U8 parsing and URL extraction
│   │   └── rewriter.go             # URL rewriting for local paths
│   ├── fetcher/
│   │   └── fetcher.go              # HTTP client with retry logic
│   └── filesystem/
│       └── filesystem.go           # File writing and path management
└── go.mod
```

### Key Components

**M3U8 Parser (`internal/downloader/m3u8.go`)**: Parses M3U8 files and extracts all URLs including segments, nested playlists, encryption keys (#EXT-X-KEY), subtitle tracks (#EXT-X-MEDIA), and initialization segments (#EXT-X-MAP). Handles both absolute and relative URL resolution.

**Downloader (`internal/downloader/downloader.go`)**: Orchestrates recursive downloads using worker pools for concurrency. Tracks visited URLs to prevent duplicates and loops. Applies include/exclude filters for file types.

**URL Rewriter (`internal/downloader/rewriter.go`)**: Rewrites absolute URLs in M3U8 files to relative local paths, preserving playlist structure. Handles both segment URLs and URI attributes in tags.

**HTTP Fetcher (`internal/fetcher/fetcher.go`)**: Uses `hashicorp/go-retryablehttp` for automatic retry with exponential backoff on network errors and 5xx responses.

**Filesystem Handler (`internal/filesystem/filesystem.go`)**: Manages URL-to-path mapping supporting both hierarchical (preserving URL structure) and flat layouts. Handles naming conflicts using URL hashing.

### CLI Flags

- `--output, -o`: Output directory (default: current directory)
- `--no-rewrite`: Keep original URLs instead of rewriting to local paths
- `--flatten`: Store all files in output directory without subdirectories
- `--include`: Comma-separated list of file extensions to download (e.g., `.m3u8,.ts`)
- `--exclude`: Comma-separated list of file extensions to skip
- `--concurrency, -c`: Number of concurrent downloads (default: 5)
- `--user-agent`: Custom User-Agent header
- `--verbose, -v`: Enable detailed logging

### Documentation Style

All public APIs use Context7-style documentation comments. See https://context7.com/golang/go for Go standard library documentation.

## Go Version

This project uses Go 1.25.3 as specified in `go.mod`.
