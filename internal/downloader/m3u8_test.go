package downloader

import (
	"net/url"
	"testing"
)

func TestParseM3U8(t *testing.T) {
	content := []byte(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXTINF:9.9,
segment1.ts
#EXTINF:9.9,
segment2.ts
#EXT-X-ENDLIST
`)

	baseURL, _ := url.Parse("https://example.com/playlist.m3u8")
	m3u8, err := ParseM3U8(content, baseURL)

	if err != nil {
		t.Fatalf("ParseM3U8 failed: %v", err)
	}

	if len(m3u8.URLs) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(m3u8.URLs))
	}

	expectedURLs := []string{
		"https://example.com/segment1.ts",
		"https://example.com/segment2.ts",
	}

	for i, expectedURL := range expectedURLs {
		if m3u8.URLs[i] != expectedURL {
			t.Errorf("URL %d: expected %s, got %s", i, expectedURL, m3u8.URLs[i])
		}
	}
}

func TestParseM3U8WithKey(t *testing.T) {
	content := []byte(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-KEY:METHOD=AES-128,URI="encryption.key"
#EXTINF:10.0,
segment.ts
`)

	baseURL, _ := url.Parse("https://example.com/path/playlist.m3u8")
	m3u8, err := ParseM3U8(content, baseURL)

	if err != nil {
		t.Fatalf("ParseM3U8 failed: %v", err)
	}

	if len(m3u8.URLs) != 2 {
		t.Errorf("Expected 2 URLs (key + segment), got %d", len(m3u8.URLs))
	}

	// Should contain the encryption key URL
	foundKey := false
	for _, u := range m3u8.URLs {
		if u == "https://example.com/path/encryption.key" {
			foundKey = true
			break
		}
	}

	if !foundKey {
		t.Error("Encryption key URL not found in parsed URLs")
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base     string
		ref      string
		expected string
	}{
		{
			base:     "https://example.com/path/playlist.m3u8",
			ref:      "segment.ts",
			expected: "https://example.com/path/segment.ts",
		},
		{
			base:     "https://example.com/path/playlist.m3u8",
			ref:      "/absolute/segment.ts",
			expected: "https://example.com/absolute/segment.ts",
		},
		{
			base:     "https://example.com/path/playlist.m3u8",
			ref:      "https://other.com/segment.ts",
			expected: "https://other.com/segment.ts",
		},
		{
			base:     "https://example.com/path/playlist.m3u8",
			ref:      "../segment.ts",
			expected: "https://example.com/segment.ts",
		},
	}

	for _, tt := range tests {
		baseURL, _ := url.Parse(tt.base)
		result := resolveURL(baseURL, tt.ref)
		if result != tt.expected {
			t.Errorf("resolveURL(%s, %s) = %s; want %s", tt.base, tt.ref, result, tt.expected)
		}
	}
}

func TestParseM3U8MasterPlaylist(t *testing.T) {
	content := []byte(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=1280000,RESOLUTION=1920x1080
high.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=640000,RESOLUTION=1280x720
medium.m3u8
`)

	baseURL, _ := url.Parse("https://example.com/master.m3u8")
	m3u8, err := ParseM3U8(content, baseURL)

	if err != nil {
		t.Fatalf("ParseM3U8 failed: %v", err)
	}

	if len(m3u8.URLs) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(m3u8.URLs))
	}

	// Check that they're marked as M3U8 files
	for _, u := range m3u8.URLs {
		if !m3u8.IsM3U8[u] {
			t.Errorf("URL %s should be marked as M3U8", u)
		}
	}
}
