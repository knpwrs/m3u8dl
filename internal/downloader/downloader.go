package downloader

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/knpwrs/m3u8dl/internal/fetcher"
	"github.com/knpwrs/m3u8dl/internal/filesystem"
)

// Downloader orchestrates the recursive downloading of M3U8 playlists.
//
// This structure manages the entire download process including:
// - Fetching M3U8 files and all referenced resources
// - Concurrent downloads with worker pools
// - Tracking visited URLs to prevent duplicates
// - Rewriting URLs in M3U8 files for local playback
// - Filtering files by type
//
// See: https://context7.com/golang/go for Go concurrency documentation
type Downloader struct {
	fetcher     *fetcher.Fetcher
	fs          *filesystem.FileSystem
	visited     map[string]bool
	visitedLock sync.Mutex
	concurrency int
	rewriteURLs bool
	include     []string // File extensions to include
	exclude     []string // File extensions to exclude
	verbose     bool
	progress    *ProgressTracker
}

// Config holds configuration for the Downloader.
type Config struct {
	OutputDir   string
	Flatten     bool
	Concurrency int
	RewriteURLs bool
	Include     []string
	Exclude     []string
	UserAgent   string
	Verbose     bool
}

// New creates a new Downloader with the given configuration.
//
// See: https://context7.com/golang/go for Go documentation
func New(cfg Config) *Downloader {
	fetcherOpts := fetcher.DefaultOptions()
	if cfg.UserAgent != "" {
		fetcherOpts.UserAgent = cfg.UserAgent
	}

	return &Downloader{
		fetcher:     fetcher.New(fetcherOpts),
		fs:          filesystem.New(cfg.OutputDir, cfg.Flatten),
		visited:     make(map[string]bool),
		concurrency: cfg.Concurrency,
		rewriteURLs: cfg.RewriteURLs,
		include:     cfg.Include,
		exclude:     cfg.Exclude,
		verbose:     cfg.Verbose,
		progress:    NewProgressTracker(true, cfg.Verbose),
	}
}

// Download starts the recursive download process from the given M3U8 URL.
//
// This method:
// 1. Downloads the initial M3U8 file
// 2. Parses it to extract all referenced URLs
// 3. Downloads all referenced files concurrently
// 4. Recursively processes any nested M3U8 playlists
// 5. Optionally rewrites URLs in M3U8 files to local paths
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - m3u8URL: The URL of the M3U8 playlist to download
//
// Returns any error encountered during the download process.
//
// See: https://context7.com/golang/go for Go context documentation
func (d *Downloader) Download(ctx context.Context, m3u8URL string) error {
	d.progress.PrintVerbose("Starting download of %s", m3u8URL)

	// Start periodic progress updates
	ticker := time.NewTicker(500 * time.Millisecond)
	done := make(chan bool)
	defer ticker.Stop()
	defer close(done)

	go func() {
		for {
			select {
			case <-ticker.C:
				d.progress.PrintProgress()
			case <-done:
				return
			}
		}
	}()

	// Download the initial M3U8 file
	if err := d.downloadM3U8(ctx, m3u8URL); err != nil {
		return fmt.Errorf("failed to download M3U8: %w", err)
	}

	// Print final summary
	d.progress.PrintSummary()
	return nil
}

// downloadM3U8 downloads an M3U8 file and all its references.
func (d *Downloader) downloadM3U8(ctx context.Context, m3u8URL string) error {
	// Check if already visited
	if d.isVisited(m3u8URL) {
		d.progress.PrintVerbose("Skipping already visited URL: %s", m3u8URL)
		return nil
	}
	d.markVisited(m3u8URL)

	// Fetch the M3U8 file
	d.progress.PrintVerbose("Fetching M3U8: %s", m3u8URL)
	content, err := d.fetcher.Fetch(ctx, m3u8URL)
	if err != nil {
		return err
	}

	// Parse the M3U8 file
	parsedURL, err := url.Parse(m3u8URL)
	if err != nil {
		return fmt.Errorf("failed to parse M3U8 URL: %w", err)
	}

	m3u8File, err := ParseM3U8(content, parsedURL)
	if err != nil {
		return fmt.Errorf("failed to parse M3U8 content: %w", err)
	}

	d.progress.PrintVerbose("Found %d URLs in M3U8", len(m3u8File.URLs))

	// Download all referenced files concurrently
	if err := d.downloadURLs(ctx, m3u8File.URLs, m3u8File.IsM3U8); err != nil {
		return err
	}

	// Rewrite URLs if enabled
	if d.rewriteURLs {
		d.progress.PrintVerbose("Rewriting URLs in M3U8 file")
		rewrittenContent, err := RewriteM3U8URLs(content, m3u8URL, d.fs)
		if err != nil {
			log.Printf("Warning: failed to rewrite URLs in %s: %v", m3u8URL, err)
			// Continue with original content
			rewrittenContent = content
		}
		content = rewrittenContent
	}

	// Write the M3U8 file
	localPath, err := d.fs.WriteFile(m3u8URL, content)
	if err != nil {
		return fmt.Errorf("failed to write M3U8 file: %w", err)
	}

	// Track progress
	d.progress.IncrementM3U8()
	d.progress.AddBytes(int64(len(content)))
	d.progress.PrintVerbose("Wrote M3U8 to %s", localPath)

	return nil
}

// downloadURLs downloads multiple URLs concurrently using a worker pool.
func (d *Downloader) downloadURLs(ctx context.Context, urls []string, isM3U8 map[string]bool) error {
	// Filter URLs
	filteredURLs := d.filterURLs(urls)
	if len(filteredURLs) == 0 {
		return nil
	}

	// Create worker pool
	urlChan := make(chan string, len(filteredURLs))
	errChan := make(chan error, len(filteredURLs))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < d.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for urlStr := range urlChan {
				if err := d.downloadURL(ctx, urlStr, isM3U8[urlStr]); err != nil {
					errChan <- fmt.Errorf("failed to download %s: %w", urlStr, err)
					return
				}
			}
		}()
	}

	// Send URLs to workers
	for _, urlStr := range filteredURLs {
		urlChan <- urlStr
	}
	close(urlChan)

	// Wait for workers to finish
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// downloadURL downloads a single URL.
func (d *Downloader) downloadURL(ctx context.Context, urlStr string, isM3U8 bool) error {
	// If it's an M3U8 file, download recursively
	// (downloadM3U8 will handle the visited check)
	if isM3U8 {
		return d.downloadM3U8(ctx, urlStr)
	}

	// Check if already visited (for non-M3U8 files)
	if d.isVisited(urlStr) {
		return nil
	}
	d.markVisited(urlStr)

	// Download as regular file
	d.progress.PrintVerbose("Downloading: %s", urlStr)
	content, err := d.fetcher.Fetch(ctx, urlStr)
	if err != nil {
		return err
	}

	localPath, err := d.fs.WriteFile(urlStr, content)
	if err != nil {
		return err
	}

	// Track progress
	d.progress.IncrementSegment(int64(len(content)))
	d.progress.PrintVerbose("Wrote to %s", localPath)

	return nil
}

// filterURLs filters URLs based on include/exclude patterns.
func (d *Downloader) filterURLs(urls []string) []string {
	filtered := make([]string, 0, len(urls))

	for _, urlStr := range urls {
		if d.shouldDownload(urlStr) {
			filtered = append(filtered, urlStr)
		}
	}

	return filtered
}

// shouldDownload checks if a URL should be downloaded based on filters.
func (d *Downloader) shouldDownload(urlStr string) bool {
	ext := strings.ToLower(filepath.Ext(urlStr))

	// Remove query parameters from extension
	if idx := strings.Index(ext, "?"); idx != -1 {
		ext = ext[:idx]
	}

	// Check exclude list first
	for _, excludeExt := range d.exclude {
		if ext == excludeExt || ext == "."+excludeExt {
			return false
		}
	}

	// If include list is empty, download everything (except excluded)
	if len(d.include) == 0 {
		return true
	}

	// Check include list
	for _, includeExt := range d.include {
		if ext == includeExt || ext == "."+includeExt {
			return true
		}
	}

	return false
}

// isVisited checks if a URL has already been visited.
func (d *Downloader) isVisited(urlStr string) bool {
	d.visitedLock.Lock()
	defer d.visitedLock.Unlock()
	return d.visited[urlStr]
}

// markVisited marks a URL as visited.
func (d *Downloader) markVisited(urlStr string) {
	d.visitedLock.Lock()
	defer d.visitedLock.Unlock()
	d.visited[urlStr] = true
}

