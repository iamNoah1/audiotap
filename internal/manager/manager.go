package manager

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

// Ensure checks for yt-dlp on PATH, then in the cache dir, then downloads it.
// It is safe to call multiple times; subsequent calls are no-ops if already resolved.
func Ensure() error {
	if resolvedPath != "" {
		return nil
	}

	// 1. Check PATH
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		resolvedPath = p
		return nil
	}

	// 2. Check cache
	cacheDir, err := getCacheDir()
	if err != nil {
		return fmt.Errorf("resolving cache dir: %w", err)
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	cached := filepath.Join(cacheDir, "yt-dlp"+ext)
	if _, err := os.Stat(cached); err == nil {
		resolvedPath = cached
		return nil
	}

	// 3. Download
	artifact, err := ytDlpArtifact()
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stderr, "audiotap: installing yt-dlp (first run)...")

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	url := downloadBaseURL + "/" + artifact
	if err := downloadFile(cached, url); err != nil {
		return fmt.Errorf("downloading yt-dlp: %w\nInstall manually: pip install yt-dlp", err)
	}

	if err := os.Chmod(cached, 0755); err != nil {
		return fmt.Errorf("making yt-dlp executable: %w", err)
	}

	fmt.Fprintln(os.Stderr, " done")
	resolvedPath = cached
	return nil
}

func downloadFile(dest, url string) error {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
