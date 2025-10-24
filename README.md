# m3u8dl

[![Test](https://github.com/knpwrs/m3u8dl/actions/workflows/test.yml/badge.svg)](https://github.com/knpwrs/m3u8dl/actions/workflows/test.yml)
[![Release](https://github.com/knpwrs/m3u8dl/actions/workflows/release.yml/badge.svg)](https://github.com/knpwrs/m3u8dl/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/knpwrs/m3u8dl/branch/main/graph/badge.svg)](https://codecov.io/gh/knpwrs/m3u8dl)

A CLI utility that downloads M3U8 playlists and all referenced files recursively.

## Features

- **Recursive Download**: Downloads M3U8 files and all referenced resources (segments, nested playlists, encryption keys, subtitles)
- **Progress Reporting**: Real-time progress updates with download speed, file counts, and elapsed time
- **URL Rewriting**: Optionally rewrites URLs in M3U8 files to local relative paths for offline playback
- **Concurrent Downloads**: Uses worker pools for fast parallel downloads with configurable concurrency
- **File Filtering**: Include or exclude specific file types using extension filters
- **Flexible Layout**: Choose between preserving URL directory structure or flattening all files
- **Retry Logic**: Automatic retry with exponential backoff for network failures
- **Deduplication**: Tracks visited URLs to avoid downloading duplicates

## Installation

### Homebrew

```bash
brew install knpwrs/tap/m3u8dl
```

### Using [`eget`](https://github.com/zyedidia/eget)

```bash
eget knpwrs/m3u8dl
```

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/knpwrs/m3u8dl/releases).

### From Source

```bash
go install github.com/knpwrs/m3u8dl@latest
```

### Build Locally

```bash
git clone https://github.com/knpwrs/m3u8dl.git
cd m3u8dl
go build -o m3u8dl .
```

## Usage

### Basic Usage

```bash
# Download M3U8 and all references to current directory
m3u8dl https://example.com/playlist.m3u8
```

### Advanced Examples

```bash
# Download to specific directory with URL rewriting
m3u8dl -o ./downloads https://example.com/playlist.m3u8

# Download without rewriting URLs (keep original absolute URLs)
m3u8dl --no-rewrite https://example.com/playlist.m3u8

# Flatten directory structure
m3u8dl --flatten -o ./downloads https://example.com/playlist.m3u8

# Download only specific file types
m3u8dl --include .m3u8,.ts https://example.com/playlist.m3u8

# Download everything except subtitles
m3u8dl --exclude .vtt,.srt https://example.com/playlist.m3u8

# Increase concurrency for faster downloads
m3u8dl -c 10 -v https://example.com/playlist.m3u8
```

## CLI Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `.` | Output directory for downloaded files |
| `--no-rewrite` | | `false` | Do not rewrite URLs in M3U8 files |
| `--flatten` | | `false` | Flatten directory structure instead of preserving URL paths |
| `--include` | | | File extensions to include (comma-separated, e.g., `.m3u8,.ts`) |
| `--exclude` | | | File extensions to exclude (comma-separated, e.g., `.vtt,.srt`) |
| `--concurrency` | `-c` | `5` | Number of concurrent downloads |
| `--user-agent` | | `m3u8dl/1.0` | Custom User-Agent header |
| `--verbose` | `-v` | `false` | Verbose logging |

## How It Works

1. **Download Initial M3U8**: Fetches the provided M3U8 URL
2. **Parse & Extract URLs**: Parses the M3U8 file and extracts all referenced URLs
3. **Concurrent Downloads**: Downloads all resources using a worker pool
4. **Recursive Processing**: Recursively processes nested M3U8 playlists
5. **URL Rewriting**: Optionally rewrites URLs to local paths
6. **Local Storage**: Saves files preserving structure or flattened

## M3U8 Support

The tool supports all standard M3U8 features:

- Master playlists with quality variants
- Media playlists with segments
- Encryption keys (`#EXT-X-KEY`)
- Alternative audio/subtitle tracks (`#EXT-X-MEDIA`)
- Initialization segments (`#EXT-X-MAP`)
- I-frame playlists (`#EXT-X-I-FRAME-STREAM-INF`)
- Both absolute and relative URLs

## Requirements

- Go 1.25.3 or later

## License

CC0 / UNLICENSE
