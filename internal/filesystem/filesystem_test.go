package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetLocalPath(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir, false)

	url := "https://example.com/path/to/file.ts"
	localPath, err := fs.GetLocalPath(url)

	if err != nil {
		t.Fatalf("GetLocalPath failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "path", "to", "file.ts")
	if localPath != expected {
		t.Errorf("Expected %s, got %s", expected, localPath)
	}

	// Second call should return cached result
	localPath2, err := fs.GetLocalPath(url)
	if err != nil {
		t.Fatalf("GetLocalPath (cached) failed: %v", err)
	}

	if localPath2 != localPath {
		t.Error("Cached path should match original path")
	}
}

func TestGetLocalPathFlattened(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir, true)

	url := "https://example.com/path/to/file.ts"
	localPath, err := fs.GetLocalPath(url)

	if err != nil {
		t.Fatalf("GetLocalPath failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "file.ts")
	if localPath != expected {
		t.Errorf("Expected %s, got %s", expected, localPath)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir, false)

	url := "https://example.com/test.txt"
	content := []byte("test content")

	localPath, err := fs.WriteFile(url, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file was written
	readContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("File content mismatch: expected %s, got %s", content, readContent)
	}
}

func TestGetRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir, false)

	fromURL := "https://example.com/path/playlist.m3u8"
	toURL := "https://example.com/path/segment.ts"

	relPath, err := fs.GetRelativePath(fromURL, toURL)
	if err != nil {
		t.Fatalf("GetRelativePath failed: %v", err)
	}

	if relPath != "segment.ts" {
		t.Errorf("Expected 'segment.ts', got %s", relPath)
	}
}

func TestGetRelativePathParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir, false)

	fromURL := "https://example.com/path/sub/playlist.m3u8"
	toURL := "https://example.com/path/segment.ts"

	relPath, err := fs.GetRelativePath(fromURL, toURL)
	if err != nil {
		t.Fatalf("GetRelativePath failed: %v", err)
	}

	if relPath != "../segment.ts" {
		t.Errorf("Expected '../segment.ts', got %s", relPath)
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir, false)

	url := "https://example.com/test.txt"

	// Should not exist initially
	exists, err := fs.FileExists(url)
	if err != nil {
		t.Fatalf("FileExists check failed: %v", err)
	}
	if exists {
		t.Error("File should not exist yet")
	}

	// Write file
	_, err = fs.WriteFile(url, []byte("test"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Should exist now
	exists, err = fs.FileExists(url)
	if err != nil {
		t.Fatalf("FileExists check failed: %v", err)
	}
	if !exists {
		t.Error("File should exist after writing")
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	fs := New(tmpDir, false)

	// Test concurrent GetLocalPath calls (tests mutex)
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			url := "https://example.com/concurrent/file.ts"
			_, err := fs.GetLocalPath(url)
			if err != nil {
				t.Errorf("Concurrent GetLocalPath failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
