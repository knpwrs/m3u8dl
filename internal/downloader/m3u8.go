package downloader

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strings"
)

// M3U8File represents a parsed M3U8 playlist file.
//
// This structure contains the original content and extracted URLs from an M3U8 playlist.
// M3U8 files are playlists used for HTTP Live Streaming (HLS) that reference media
// segments and other resources.
//
// See: https://context7.com/golang/go for Go documentation
type M3U8File struct {
	Content  []byte
	BaseURL  *url.URL
	URLs     []string
	IsM3U8   map[string]bool // Track which URLs are M3U8 files
}

// ParseM3U8 parses an M3U8 file and extracts all referenced URLs.
//
// This function processes both master playlists (which reference other playlists)
// and media playlists (which reference media segments). It handles:
// - Media segments (.ts, .mp4, .aac, etc.)
// - Nested M3U8 playlists
// - Encryption keys (#EXT-X-KEY)
// - Subtitle/caption files (#EXT-X-MEDIA)
// - Map files (#EXT-X-MAP)
//
// Parameters:
//   - content: The raw M3U8 file content
//   - baseURL: The URL from which this M3U8 was fetched, used for resolving relative URLs
//
// Returns the parsed M3U8File structure containing all extracted URLs.
//
// See: https://context7.com/golang/go for Go documentation
func ParseM3U8(content []byte, baseURL *url.URL) (*M3U8File, error) {
	m3u8 := &M3U8File{
		Content:  content,
		BaseURL:  baseURL,
		URLs:     make([]string, 0),
		IsM3U8:   make(map[string]bool),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments that don't contain URLs
		if line == "" || (strings.HasPrefix(line, "#") && !containsURL(line)) {
			continue
		}

		// Extract URLs from tag lines
		if strings.HasPrefix(line, "#") {
			urls := extractURLsFromTag(line)
			for _, u := range urls {
				resolved := resolveURL(baseURL, u)
				m3u8.URLs = append(m3u8.URLs, resolved)
				// Mark encryption keys and other resources
				if !strings.HasSuffix(u, ".m3u8") {
					m3u8.IsM3U8[resolved] = false
				}
			}
			continue
		}

		// Non-comment lines are either segment URLs or playlist URLs
		if !strings.HasPrefix(line, "#") {
			resolved := resolveURL(baseURL, line)
			m3u8.URLs = append(m3u8.URLs, resolved)
			// Check if it's likely an M3U8 file
			m3u8.IsM3U8[resolved] = strings.HasSuffix(line, ".m3u8") || strings.Contains(line, ".m3u8?")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning M3U8 file: %w", err)
	}

	return m3u8, nil
}

// containsURL checks if a tag line contains a URL.
func containsURL(line string) bool {
	urlTags := []string{"EXT-X-KEY", "EXT-X-MEDIA", "EXT-X-MAP", "EXT-X-I-FRAME-STREAM-INF"}
	for _, tag := range urlTags {
		if strings.Contains(line, tag) {
			return true
		}
	}
	return false
}

// extractURLsFromTag extracts URLs from M3U8 tag lines.
//
// Handles various M3U8 tags that reference URLs:
// - #EXT-X-KEY:URI="url" - Encryption keys
// - #EXT-X-MEDIA:URI="url" - Alternative audio/subtitle tracks
// - #EXT-X-MAP:URI="url" - Initialization segments
// - #EXT-X-I-FRAME-STREAM-INF:URI="url" - I-frame playlists
func extractURLsFromTag(line string) []string {
	urls := make([]string, 0)

	// Extract URI attribute values
	uriPrefix := "URI=\""
	for {
		idx := strings.Index(line, uriPrefix)
		if idx == -1 {
			break
		}
		line = line[idx+len(uriPrefix):]
		endIdx := strings.Index(line, "\"")
		if endIdx == -1 {
			break
		}
		urls = append(urls, line[:endIdx])
		line = line[endIdx+1:]
	}

	return urls
}

// resolveURL resolves a potentially relative URL against a base URL.
//
// This handles three cases:
// 1. Absolute URLs (http://, https://) - returned as-is
// 2. Absolute paths (/path/to/file) - combined with base scheme and host
// 3. Relative paths (../path or file.ts) - resolved relative to base URL's path
//
// See: https://context7.com/golang/go for Go URL handling documentation
func resolveURL(base *url.URL, ref string) string {
	// Parse the reference URL
	refURL, err := url.Parse(ref)
	if err != nil {
		// If parsing fails, treat as relative path
		return resolveRelativePath(base, ref)
	}

	// If it's already absolute, return as-is
	if refURL.IsAbs() {
		return ref
	}

	// Resolve relative URL against base
	resolved := base.ResolveReference(refURL)
	return resolved.String()
}

// resolveRelativePath resolves a relative path against a base URL.
func resolveRelativePath(base *url.URL, relativePath string) string {
	// Start with the base URL's directory
	basePath := path.Dir(base.Path)

	// Join with relative path
	resolved := path.Join(basePath, relativePath)

	// Construct full URL
	result := &url.URL{
		Scheme: base.Scheme,
		Host:   base.Host,
		Path:   resolved,
	}

	return result.String()
}
