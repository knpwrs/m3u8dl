package filesystem

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// FileSystem handles file writing and path management for downloaded files.
//
// This structure manages the mapping between URLs and local file paths,
// supporting both hierarchical (preserving URL structure) and flat
// (all files in one directory) layouts.
//
// See: https://context7.com/golang/go for Go file I/O documentation
type FileSystem struct {
	outputDir   string
	flatten     bool
	urlToPath   map[string]string // Cache URL to file path mappings
	pathsInUse  map[string]bool   // Track used paths to avoid collisions
	mu          sync.Mutex        // Protects urlToPath and pathsInUse
}

// New creates a new FileSystem handler.
//
// Parameters:
//   - outputDir: The base directory where files will be written
//   - flatten: If true, all files are written to outputDir without subdirectories
//
// See: https://context7.com/golang/go for Go documentation
func New(outputDir string, flatten bool) *FileSystem {
	return &FileSystem{
		outputDir:  outputDir,
		flatten:    flatten,
		urlToPath:  make(map[string]string),
		pathsInUse: make(map[string]bool),
	}
}

// GetLocalPath returns the local file path for a given URL.
//
// This method ensures consistent path mapping and handles naming conflicts.
// For hierarchical mode, it preserves the URL's path structure.
// For flat mode, it uses the filename with conflict resolution.
//
// Parameters:
//   - urlStr: The URL to map to a local path
//
// Returns the local file path where the URL's content should be stored.
func (fs *FileSystem) GetLocalPath(urlStr string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Return cached path if available
	if localPath, exists := fs.urlToPath[urlStr]; exists {
		return localPath, nil
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %s: %w", urlStr, err)
	}

	var localPath string

	if fs.flatten {
		// Extract filename from URL
		filename := path.Base(parsedURL.Path)
		if filename == "" || filename == "." || filename == "/" {
			// Generate filename from URL hash if no clear filename
			filename = generateFilenameFromURL(urlStr)
		}

		localPath = filepath.Join(fs.outputDir, filename)

		// Handle naming conflicts
		if fs.pathsInUse[localPath] {
			localPath = fs.resolveConflict(localPath, urlStr)
		}
	} else {
		// Preserve URL path structure
		// Remove leading slash and convert to local path
		urlPath := strings.TrimPrefix(parsedURL.Path, "/")
		localPath = filepath.Join(fs.outputDir, filepath.FromSlash(urlPath))
	}

	fs.urlToPath[urlStr] = localPath
	fs.pathsInUse[localPath] = true

	return localPath, nil
}

// WriteFile writes content to the local path for the given URL.
//
// This method creates any necessary parent directories and writes the file
// atomically by writing to a temporary file first, then renaming.
//
// Parameters:
//   - urlStr: The URL whose content is being written
//   - content: The file content to write
//
// Returns the local path where the file was written and any error encountered.
//
// See: https://context7.com/golang/go for Go file operations
func (fs *FileSystem) WriteFile(urlStr string, content []byte) (string, error) {
	localPath, err := fs.GetLocalPath(urlStr)
	if err != nil {
		return "", err
	}

	// Create parent directories
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write to temporary file first
	tmpPath := localPath + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", tmpPath, err)
	}

	// Rename to final path (atomic on most systems)
	if err := os.Rename(tmpPath, localPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return "", fmt.Errorf("failed to rename %s to %s: %w", tmpPath, localPath, err)
	}

	return localPath, nil
}

// GetRelativePath returns the relative path from one URL's local path to another.
//
// This is used for URL rewriting in M3U8 files - converting absolute URLs to
// relative paths that work in the local file system.
//
// Parameters:
//   - fromURL: The URL of the M3U8 file containing the reference
//   - toURL: The URL being referenced
//
// Returns the relative path from fromURL's file to toURL's file.
func (fs *FileSystem) GetRelativePath(fromURL, toURL string) (string, error) {
	fromPath, err := fs.GetLocalPath(fromURL)
	if err != nil {
		return "", err
	}

	toPath, err := fs.GetLocalPath(toURL)
	if err != nil {
		return "", err
	}

	// Get the directory containing the from file
	fromDir := filepath.Dir(fromPath)

	// Calculate relative path
	relPath, err := filepath.Rel(fromDir, toPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Convert back to forward slashes for URLs
	relPath = filepath.ToSlash(relPath)

	return relPath, nil
}

// resolveConflict handles filename conflicts by appending a hash.
func (fs *FileSystem) resolveConflict(originalPath, urlStr string) string {
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)

	// Use URL hash to create unique filename
	hash := generateHashFromURL(urlStr)
	newPath := fmt.Sprintf("%s_%s%s", base, hash[:8], ext)

	// If still conflicts (very unlikely), keep appending
	counter := 1
	for fs.pathsInUse[newPath] {
		newPath = fmt.Sprintf("%s_%s_%d%s", base, hash[:8], counter, ext)
		counter++
	}

	return newPath
}

// generateFilenameFromURL creates a filename from a URL when no filename is present.
func generateFilenameFromURL(urlStr string) string {
	hash := generateHashFromURL(urlStr)
	return hash[:16] + ".bin"
}

// generateHashFromURL creates a hash of a URL for unique identification.
func generateHashFromURL(urlStr string) string {
	hasher := sha256.New()
	hasher.Write([]byte(urlStr))
	return hex.EncodeToString(hasher.Sum(nil))
}

// FileExists checks if a file already exists at the local path for a URL.
//
// This can be used to skip re-downloading files that already exist.
func (fs *FileSystem) FileExists(urlStr string) (bool, error) {
	localPath, err := fs.GetLocalPath(urlStr)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(localPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
