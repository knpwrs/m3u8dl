package main

import "github.com/knpwrs/m3u8dl/cmd"

// main is the entry point for the m3u8dl CLI application.
//
// This application downloads M3U8 playlists and all referenced files recursively,
// with support for URL rewriting, file filtering, and concurrent downloads.
//
// See: https://context7.com/golang/go for Go documentation
func main() {
	cmd.Execute()
}
