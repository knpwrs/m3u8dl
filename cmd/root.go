package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/knpwrs/m3u8dl/internal/downloader"
	"github.com/spf13/cobra"
)

var (
	outputDir   string
	noRewrite   bool
	flatten     bool
	include     []string
	exclude     []string
	concurrency int
	userAgent   string
	verbose     bool
)

// rootCmd represents the base command when called without any subcommands.
//
// This CLI tool downloads M3U8 playlists and all referenced files recursively,
// optionally rewriting URLs to create a fully local copy that can be played
// without network access.
//
// See: https://context7.com/golang/go for Go documentation
var rootCmd = &cobra.Command{
	Use:   "m3u8dl [URL]",
	Short: "Download and mirror M3U8 playlists",
	Long: `m3u8dl is a CLI utility that downloads M3U8 playlists and all referenced files recursively.

It downloads the M3U8 file and all resources it references (segments, nested playlists,
encryption keys, subtitles, etc.), and can optionally rewrite URLs to create a fully
local copy that can be played without network access.`,
	Example: `  # Download M3U8 and all references to current directory
  m3u8dl https://example.com/playlist.m3u8

  # Download to specific directory with URL rewriting
  m3u8dl -o ./downloads https://example.com/playlist.m3u8

  # Download without rewriting URLs (keep original absolute URLs)
  m3u8dl --no-rewrite https://example.com/playlist.m3u8

  # Flatten directory structure
  m3u8dl --flatten -o ./downloads https://example.com/playlist.m3u8

  # Download only specific file types
  m3u8dl --include .m3u8,.ts https://example.com/playlist.m3u8

  # Download everything except subtitles
  m3u8dl --exclude .vtt,.srt https://example.com/playlist.m3u8`,
	Args: cobra.ExactArgs(1),
	RunE: runDownload,
}

// Execute adds all child commands to the root command and sets flags appropriately.
//
// This is called by main.main(). It only needs to happen once to the rootCmd.
//
// See: https://context7.com/golang/go for Go documentation
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for downloaded files")
	rootCmd.Flags().BoolVar(&noRewrite, "no-rewrite", false, "Do not rewrite URLs in M3U8 files (keep original URLs)")
	rootCmd.Flags().BoolVar(&flatten, "flatten", false, "Flatten directory structure instead of preserving URL paths")
	rootCmd.Flags().StringSliceVar(&include, "include", []string{}, "File extensions to include (comma-separated, e.g., .m3u8,.ts)")
	rootCmd.Flags().StringSliceVar(&exclude, "exclude", []string{}, "File extensions to exclude (comma-separated, e.g., .vtt,.srt)")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 5, "Number of concurrent downloads")
	rootCmd.Flags().StringVar(&userAgent, "user-agent", "", "Custom User-Agent header")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}

// runDownload is the main execution function for the root command.
func runDownload(cmd *cobra.Command, args []string) error {
	m3u8URL := args[0]

	// Validate URL
	if !strings.HasPrefix(m3u8URL, "http://") && !strings.HasPrefix(m3u8URL, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// Normalize include/exclude extensions
	include = normalizeExtensions(include)
	exclude = normalizeExtensions(exclude)

	// Create downloader configuration
	cfg := downloader.Config{
		OutputDir:   outputDir,
		Flatten:     flatten,
		Concurrency: concurrency,
		RewriteURLs: !noRewrite,
		Include:     include,
		Exclude:     exclude,
		UserAgent:   userAgent,
		Verbose:     verbose,
	}

	// Create and run downloader
	dl := downloader.New(cfg)
	ctx := context.Background()

	if verbose {
		fmt.Printf("Starting download of %s\n", m3u8URL)
		fmt.Printf("Output directory: %s\n", outputDir)
		fmt.Printf("URL rewriting: %v\n", !noRewrite)
		fmt.Printf("Flatten structure: %v\n", flatten)
		fmt.Printf("Concurrency: %d\n", concurrency)
		if len(include) > 0 {
			fmt.Printf("Include extensions: %v\n", include)
		}
		if len(exclude) > 0 {
			fmt.Printf("Exclude extensions: %v\n", exclude)
		}
		fmt.Println()
	}

	if err := dl.Download(ctx, m3u8URL); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Println("Download completed successfully")
	return nil
}

// normalizeExtensions ensures all extensions start with a dot.
func normalizeExtensions(exts []string) []string {
	normalized := make([]string, len(exts))
	for i, ext := range exts {
		ext = strings.TrimSpace(ext)
		if ext != "" && !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		normalized[i] = ext
	}
	return normalized
}
