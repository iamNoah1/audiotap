package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// resolvedPath holds the yt-dlp binary path after Ensure() succeeds.
var resolvedPath string

// downloadBaseURL is the base URL for yt-dlp release artifacts.
// Overridden in tests via SetDownloadBaseURL.
var downloadBaseURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download"

// SetDownloadBaseURL overrides the download URL — for testing only.
func SetDownloadBaseURL(u string) { downloadBaseURL = u }

// overrideCacheDir overrides getCacheDir — for testing only.
var overrideCacheDir string

// SetCacheDir overrides the cache directory — for testing only.
func SetCacheDir(dir string) { overrideCacheDir = dir }

// BinaryPath returns the resolved yt-dlp path. Call Ensure() first.
func BinaryPath() string { return resolvedPath }

func getCacheDir() (string, error) {
	if overrideCacheDir != "" {
		return overrideCacheDir, nil
	}
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			return "", fmt.Errorf("%%APPDATA%% not set")
		}
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home dir: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "audiotap", "bin"), nil
}

func ytDlpArtifact() (string, error) {
	key := runtime.GOOS + "/" + runtime.GOARCH
	artifacts := map[string]string{
		"linux/amd64":   "yt-dlp_linux",
		"linux/arm64":   "yt-dlp_linux_aarch64",
		"darwin/amd64":  "yt-dlp_macos_legacy",
		"darwin/arm64":  "yt-dlp_macos",
		"windows/amd64": "yt-dlp.exe",
	}
	if a, ok := artifacts[key]; ok {
		return a, nil
	}
	return "", fmt.Errorf("unsupported platform %s — install yt-dlp manually: pip install yt-dlp", key)
}
