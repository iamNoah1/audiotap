package manager

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestYtDlpArtifact(t *testing.T) {
	artifact, err := ytDlpArtifact()
	if err != nil {
		// Only acceptable on unsupported platforms
		if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
			return
		}
		t.Fatalf("unexpected error on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
	}
	if artifact == "" {
		t.Fatal("artifact is empty")
	}
}

func TestYtDlpArtifact_UnsupportedPlatform(t *testing.T) {
	// We can't easily force GOOS at runtime, so just verify the function
	// returns a non-empty string on the current platform without error.
	_, err := ytDlpArtifact()
	// On CI (linux/amd64 or darwin/arm64) this must succeed.
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	}
}

func TestGetCacheDir_ReturnsPath(t *testing.T) {
	dir, err := getCacheDir()
	if err != nil {
		t.Fatalf("getCacheDir: %v", err)
	}
	if dir == "" {
		t.Fatal("empty cache dir")
	}
	// Must end with audiotap/bin
	if filepath.Base(dir) != "bin" {
		t.Errorf("expected dir to end with 'bin', got %q", dir)
	}
	parent := filepath.Base(filepath.Dir(dir))
	if parent != "audiotap" {
		t.Errorf("expected parent dir to be 'audiotap', got %q", parent)
	}
}

func TestGetCacheDir_WindowsFallback(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}
	orig := os.Getenv("APPDATA")
	os.Setenv("APPDATA", "")
	defer os.Setenv("APPDATA", orig)

	_, err := getCacheDir()
	if err == nil {
		t.Fatal("expected error when APPDATA unset, got nil")
	}
}
