package manager

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestGetCacheDir_Override(t *testing.T) {
	want := t.TempDir()
	SetCacheDir(want)
	defer SetCacheDir("")

	got, err := getCacheDir()
	if err != nil {
		t.Fatalf("getCacheDir: %v", err)
	}
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
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

func TestEnsure_CacheHit(t *testing.T) {
	tmp := t.TempDir()
	SetCacheDir(tmp)
	defer SetCacheDir("")
	resolvedPath = "" // reset package state

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	// Create a fake binary in the cache dir
	fake := filepath.Join(tmp, "yt-dlp"+ext)
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("creating fake binary: %v", err)
	}

	// Override PATH so exec.LookPath cannot find a system yt-dlp,
	// forcing Ensure() to fall through to the cache check.
	emptyDir := t.TempDir()
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", emptyDir)
	defer os.Setenv("PATH", origPath)

	// Ensure must NOT make any network call — it finds the cached binary
	// We point downloads at an invalid URL to confirm no request is made
	origURL := downloadBaseURL
	SetDownloadBaseURL("http://127.0.0.1:1") // connection refused
	defer SetDownloadBaseURL(origURL)

	if err := Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if BinaryPath() != fake {
		t.Errorf("expected %q, got %q", fake, BinaryPath())
	}
}

func TestEnsure_Downloads(t *testing.T) {
	// Serve a minimal fake binary from a test HTTP server
	served := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
		fmt.Fprint(w, "#!/bin/sh\necho fake-yt-dlp\n")
	}))
	defer ts.Close()

	tmp := t.TempDir()
	SetCacheDir(tmp)
	defer SetCacheDir("")
	SetDownloadBaseURL(ts.URL)
	defer SetDownloadBaseURL("https://github.com/yt-dlp/yt-dlp/releases/latest/download")
	resolvedPath = ""

	// Temporarily make exec.LookPath fail by pointing PATH to an empty dir
	emptyDir := t.TempDir()
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", emptyDir)
	defer os.Setenv("PATH", origPath)

	if err := Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if !served {
		t.Error("expected HTTP server to be hit")
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	expected := filepath.Join(tmp, "yt-dlp"+ext)
	if BinaryPath() != expected {
		t.Errorf("expected %q, got %q", expected, BinaryPath())
	}
	info, err := os.Stat(expected)
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		t.Error("binary is not executable")
	}
}
