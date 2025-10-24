package downloader

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/knpwrs/m3u8dl/internal/filesystem"
)

// RewriteM3U8URLs rewrites all URLs in an M3U8 file to local relative paths.
//
// This function processes an M3U8 file and converts all absolute URLs to
// relative paths that reference the locally downloaded files. This allows
// the downloaded M3U8 playlist to be played locally without network access.
//
// Parameters:
//   - content: The original M3U8 file content
//   - sourceURL: The URL where this M3U8 was downloaded from
//   - fs: The filesystem handler that manages path mappings
//
// Returns the rewritten M3U8 content with local relative paths.
//
// See: https://context7.com/golang/go for Go documentation
func RewriteM3U8URLs(content []byte, sourceURL string, fs *filesystem.FileSystem) ([]byte, error) {
	var output bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		rewrittenLine := line

		// Check if line contains URLs that need rewriting
		if strings.HasPrefix(line, "#") {
			// Handle tag lines with URI attributes
			if containsURI(line) {
				var err error
				rewrittenLine, err = rewriteTagLine(line, sourceURL, fs)
				if err != nil {
					// On error, keep original line
					rewrittenLine = line
				}
			}
		} else if line != "" && !strings.HasPrefix(line, "#") {
			// Non-comment, non-empty lines are segment or playlist URLs
			rewrittenLine = rewriteSegmentLine(line, sourceURL, fs)
		}

		output.WriteString(rewrittenLine)
		output.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning M3U8 file: %w", err)
	}

	return output.Bytes(), nil
}

// containsURI checks if a line contains URI attributes that need rewriting.
func containsURI(line string) bool {
	return strings.Contains(line, "URI=\"")
}

// rewriteTagLine rewrites URLs in M3U8 tag lines (e.g., #EXT-X-KEY, #EXT-X-MEDIA).
func rewriteTagLine(line, sourceURL string, fs *filesystem.FileSystem) (string, error) {
	result := line

	// Find and replace all URI="..." patterns
	for {
		uriStart := strings.Index(result, "URI=\"")
		if uriStart == -1 {
			break
		}

		// Find the end quote
		uriValueStart := uriStart + len("URI=\"")
		uriEnd := strings.Index(result[uriValueStart:], "\"")
		if uriEnd == -1 {
			break
		}

		// Extract the URL
		absoluteURL := result[uriValueStart : uriValueStart+uriEnd]

		// Resolve URL (in case it's relative in the original M3U8)
		m3u8File, err := ParseM3U8([]byte{}, mustParseURL(sourceURL))
		if err == nil {
			absoluteURL = resolveURL(m3u8File.BaseURL, absoluteURL)
		}

		// Get relative path
		relativePath, err := fs.GetRelativePath(sourceURL, absoluteURL)
		if err != nil {
			// If we can't get relative path, skip this URL
			result = result[uriValueStart+uriEnd:]
			continue
		}

		// Replace the URL with relative path
		before := result[:uriValueStart]
		after := result[uriValueStart+uriEnd:]
		result = before + relativePath + after

		// Move past this replacement to find next URI
		result = before + relativePath + after
		break // Process one URI at a time to avoid infinite loops
	}

	return result, nil
}

// rewriteSegmentLine rewrites URLs in segment/playlist lines.
func rewriteSegmentLine(line, sourceURL string, fs *filesystem.FileSystem) string {
	// The line is a URL (absolute or relative)
	// First resolve it to absolute
	m3u8File, err := ParseM3U8([]byte{}, mustParseURL(sourceURL))
	if err != nil {
		return line
	}

	absoluteURL := resolveURL(m3u8File.BaseURL, line)

	// Get relative path
	relativePath, err := fs.GetRelativePath(sourceURL, absoluteURL)
	if err != nil {
		return line
	}

	return relativePath
}

// mustParseURL parses a URL and panics on error (helper for internal use).
func mustParseURL(urlStr string) *url.URL {
	u, err := url.Parse(urlStr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse URL %s: %v", urlStr, err))
	}
	return u
}
