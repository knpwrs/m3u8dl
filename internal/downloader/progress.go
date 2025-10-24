package downloader

import (
	"fmt"
	"sync"
	"time"
)

// ProgressTracker tracks download progress and provides formatted output.
//
// This structure maintains statistics about the download process including
// total files, downloaded files, bytes transferred, and download speed.
//
// See: https://context7.com/golang/go for Go documentation
type ProgressTracker struct {
	mu sync.Mutex

	// File counts
	totalFiles      int
	downloadedFiles int
	m3u8Files       int
	segmentFiles    int

	// Byte counts
	totalBytes      int64
	downloadedBytes int64

	// Timing
	startTime   time.Time
	lastUpdate  time.Time
	lastBytes   int64

	// Display
	enabled bool
	verbose bool
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker(enabled, verbose bool) *ProgressTracker {
	return &ProgressTracker{
		enabled:   enabled,
		verbose:   verbose,
		startTime: time.Now(),
	}
}

// SetTotalFiles sets the total number of files to download.
func (p *ProgressTracker) SetTotalFiles(total int) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalFiles = total
}

// IncrementM3U8 increments the M3U8 file counter.
func (p *ProgressTracker) IncrementM3U8() {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m3u8Files++
	p.downloadedFiles++
}

// IncrementSegment increments the segment file counter.
func (p *ProgressTracker) IncrementSegment(bytes int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.segmentFiles++
	p.downloadedFiles++
	p.downloadedBytes += bytes
}

// AddBytes adds to the downloaded bytes counter.
func (p *ProgressTracker) AddBytes(bytes int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.downloadedBytes += bytes
}

// PrintProgress prints a formatted progress update.
func (p *ProgressTracker) PrintProgress() {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(p.startTime)

	// Calculate download speed
	var speed float64
	if elapsed.Seconds() > 0 {
		speed = float64(p.downloadedBytes) / elapsed.Seconds()
	}

	// Format bytes
	downloadedStr := formatBytes(p.downloadedBytes)
	speedStr := formatBytes(int64(speed)) + "/s"

	// Build progress message
	var msg string
	if p.totalFiles > 0 {
		percentage := float64(p.downloadedFiles) / float64(p.totalFiles) * 100
		msg = fmt.Sprintf("Progress: %d/%d files (%.1f%%) | %s downloaded | %s | Elapsed: %s",
			p.downloadedFiles, p.totalFiles, percentage, downloadedStr, speedStr, formatDuration(elapsed))
	} else {
		msg = fmt.Sprintf("Progress: %d files | %s downloaded | %s | Elapsed: %s",
			p.downloadedFiles, downloadedStr, speedStr, formatDuration(elapsed))
	}

	fmt.Printf("\r%-120s", msg)
}

// PrintSummary prints a final summary of the download.
func (p *ProgressTracker) PrintSummary() {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	fmt.Println() // New line after progress bar
	fmt.Println("\n" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                              Download Complete")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	elapsed := time.Since(p.startTime)
	avgSpeed := float64(p.downloadedBytes) / elapsed.Seconds()

	fmt.Printf("  Total Files Downloaded: %d\n", p.downloadedFiles)
	fmt.Printf("    • M3U8 Playlists:      %d\n", p.m3u8Files)
	fmt.Printf("    • Media Segments:      %d\n", p.segmentFiles)
	fmt.Printf("\n")
	fmt.Printf("  Total Data:             %s\n", formatBytes(p.downloadedBytes))
	fmt.Printf("  Average Speed:          %s/s\n", formatBytes(int64(avgSpeed)))
	fmt.Printf("  Time Elapsed:           %s\n", formatDuration(elapsed))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// PrintVerbose prints a verbose message if verbose mode is enabled.
func (p *ProgressTracker) PrintVerbose(format string, args ...interface{}) {
	if !p.enabled || !p.verbose {
		return
	}
	// Clear the progress line before printing verbose output
	fmt.Printf("\r%-120s\r", "")
	fmt.Printf(format+"\n", args...)
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
