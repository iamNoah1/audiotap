# Zero-Setup Install + Single-Download Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `audiotap` work out of the box after `go install` — no manual yt-dlp or ffmpeg setup — and show a live progress bar for single-URL downloads.

**Architecture:** A new `internal/manager` package auto-installs yt-dlp on first run and exposes `BinaryPath()` for the downloader to use. A new `progressWriter` in `downloader/progress.go` parses yt-dlp's stderr output to drive a progress bar. The default audio format changes from `mp3` to `opus` (YouTube-native, no ffmpeg needed).

**Tech Stack:** Go 1.22, `github.com/schollz/progressbar/v3` (already in `go.mod`), `net/http` for binary download, standard library only for manager.

---

## File Map

| File | Status | Responsibility |
|---|---|---|
| `internal/manager/manager.go` | **Create** | yt-dlp resolution: PATH → cache → download |
| `internal/manager/manager_test.go` | **Create** | Unit tests for Ensure(), BinaryPath(), platform mapping |
| `downloader/progress.go` | **Create** | `progressWriter` io.Writer + `DownloadWithProgress` |
| `downloader/progress_test.go` | **Create** | Unit tests for progressWriter line parsing |
| `downloader/downloader.go` | **Modify** | Use `manager.BinaryPath()` instead of `"yt-dlp"`; remove yt-dlp PATH check |
| `cmd/root.go` | **Modify** | Call `manager.Ensure()` at startup; default format `opus`; use `DownloadWithProgress` for single URL |
| `README.md` | **Modify** | Remove ffmpeg/yt-dlp from requirements; update default format; add auto-install note |

---

## Task 1: manager package — platform mapping + cache dir

**Files:**
- Create: `internal/manager/manager.go`
- Create: `internal/manager/manager_test.go`

- [ ] **Step 1: Write failing tests for `ytDlpArtifact` and `getCacheDir`**

Create `internal/manager/manager_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd /path/to/audiotap
go test ./internal/manager/... -v
```

Expected: compilation error — package does not exist yet.

- [ ] **Step 3: Create `internal/manager/manager.go` with platform mapping + cache dir**

```go
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
```

- [ ] **Step 4: Run tests — expect them to pass**

```bash
go test ./internal/manager/... -v -run TestYtDlpArtifact
go test ./internal/manager/... -v -run TestGetCacheDir
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/manager/manager.go internal/manager/manager_test.go
git commit -m "feat(manager): add platform mapping and cache dir resolution"
```

---

## Task 2: manager package — `Ensure()` with download

**Files:**
- Modify: `internal/manager/manager.go`
- Modify: `internal/manager/manager_test.go`

- [ ] **Step 1: Write failing tests for `Ensure()`**

Append to `internal/manager/manager_test.go`:

```go
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
```

Add missing imports at the top of `manager_test.go` (replace the existing import block):

```go
import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/manager/... -v -run TestEnsure
```

Expected: compile error — `Ensure` and `downloadFile` not defined yet.

- [ ] **Step 3: Implement `Ensure()` and `downloadFile()` in `manager.go`**

Append to `internal/manager/manager.go`:

```go
import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)
```

Replace the existing import block with the above, then append these functions to the file:

```go
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
```

- [ ] **Step 4: Run tests — expect them to pass**

```bash
go test ./internal/manager/... -v
```

Expected: all PASS. If `TestEnsure_Downloads` fails due to yt-dlp being on PATH in your environment, that is correct behavior (it uses the real binary and skips the download).

- [ ] **Step 5: Commit**

```bash
git add internal/manager/manager.go internal/manager/manager_test.go
git commit -m "feat(manager): implement Ensure() with auto-download of yt-dlp"
```

---

## Task 3: Wire manager into `downloader/downloader.go`

**Files:**
- Modify: `downloader/downloader.go`

- [ ] **Step 1: Read the current file**

Read `downloader/downloader.go` in full before editing.

- [ ] **Step 2: Update imports, remove `ErrYtDlpNotFound`, use `manager.BinaryPath()`**

Replace the full contents of `downloader/downloader.go` with:

```go
package downloader

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/iamNoah1/audiotap/internal/manager"
)

// ErrFfmpegNotFound is returned when ffmpeg is not on PATH and mp3/wav conversion is requested.
var ErrFfmpegNotFound = errors.New("ffmpeg not found. Install via brew install ffmpeg (macOS) or apt install ffmpeg (Linux)")

// ErrInvalidURL is returned when the URL does not look like a YouTube URL.
var ErrInvalidURL = errors.New("invalid YouTube URL")

// ErrDownloadFailed is returned when yt-dlp exits with a non-zero status.
var ErrDownloadFailed = errors.New("download failed. Check your connection or try again")

// Config holds parameters for a single download.
type Config struct {
	OutputDir string
	Format    string // mp3, opus, wav
}

// Download extracts audio from a YouTube URL and returns the output file path.
func Download(rawURL string, cfg Config) (string, error) {
	if err := validateURL(rawURL); err != nil {
		return "", err
	}
	if err := checkDependencies(cfg.Format); err != nil {
		return "", err
	}

	outDir := cfg.OutputDir
	if outDir == "" {
		var err error
		outDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
	}

	outputTemplate := outDir + "/%(title)s.%(ext)s"

	args := []string{
		"--no-check-certificates",
		"-x",
		"--audio-format", cfg.Format,
		"--audio-quality", "0",
		"-o", outputTemplate,
		"--print", "after_move:filepath",
		rawURL,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(manager.BinaryPath(), args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "network") || strings.Contains(msg, "Unable to download") {
			return "", fmt.Errorf("%w\nTip: make sure the URL is quoted to avoid shell escaping issues", ErrDownloadFailed)
		}
		return "", fmt.Errorf("%w\n%s", ErrDownloadFailed, msg)
	}

	outFile := strings.TrimSpace(stdout.String())
	return outFile, nil
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	if host != "youtube.com" && host != "youtu.be" {
		return ErrInvalidURL
	}
	return nil
}

func checkDependencies(format string) error {
	if format == "mp3" || format == "wav" {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			return ErrFfmpegNotFound
		}
	}
	return nil
}
```

- [ ] **Step 3: Build to confirm it compiles**

```bash
go build ./...
```

Expected: exits 0, no errors.

- [ ] **Step 4: Run existing downloader tests**

```bash
go test ./downloader/... -v
```

Expected: all existing tests PASS (none should reference `ErrYtDlpNotFound` — if they do, remove those test cases).

- [ ] **Step 5: Commit**

```bash
git add downloader/downloader.go
git commit -m "feat(downloader): use manager.BinaryPath() instead of hardcoded yt-dlp"
```

---

## Task 4: Progress writer

**Files:**
- Create: `downloader/progress.go`
- Create: `downloader/progress_test.go`

- [ ] **Step 1: Write failing tests for `progressWriter`**

Create `downloader/progress_test.go`:

```go
package downloader

import (
	"io"
	"strings"
	"testing"

	"github.com/schollz/progressbar/v3"
)

func newTestBar() *progressbar.ProgressBar {
	return progressbar.NewOptions(100,
		progressbar.OptionSetWriter(io.Discard),
		progressbar.OptionSilentErrors(),
	)
}

func TestProgressWriter_ParsesPercentage(t *testing.T) {
	bar := newTestBar()
	pw := &progressWriter{bar: bar}

	input := "[download]  23.4% of   4.20MiB at  1.23MiB/s ETA 00:02\n"
	if _, err := pw.Write([]byte(input)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := int(bar.State().CurrentPercent * 100)
	if got < 22 || got > 24 {
		t.Errorf("expected ~23, got %d", got)
	}
}

func TestProgressWriter_IgnoresNonProgressLines(t *testing.T) {
	bar := newTestBar()
	pw := &progressWriter{bar: bar}

	// These lines should not advance the bar
	lines := []string{
		"[youtube] Extracting URL: https://youtu.be/abc\n",
		"[info] Writing video subtitles to: video.en.vtt\n",
		"[ExtractAudio] Destination: video.opus\n",
	}
	for _, l := range lines {
		if _, err := pw.Write([]byte(l)); err != nil {
			t.Fatalf("Write(%q): %v", l, err)
		}
	}

	if bar.State().CurrentPercent != 0 {
		t.Errorf("expected 0%%, got %v%%", bar.State().CurrentPercent*100)
	}
}

func TestProgressWriter_HandlesCarriageReturn(t *testing.T) {
	bar := newTestBar()
	pw := &progressWriter{bar: bar}

	// yt-dlp uses \r without --newline; we handle both
	input := "[download]  50.0% of   4.20MiB at  2.00MiB/s ETA 00:01\r"
	if _, err := pw.Write([]byte(input)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := int(bar.State().CurrentPercent * 100)
	if got < 49 || got > 51 {
		t.Errorf("expected ~50, got %d", got)
	}
}

func TestProgressWriter_HandlesMultipleLines(t *testing.T) {
	bar := newTestBar()
	pw := &progressWriter{bar: bar}

	input := strings.Join([]string{
		"[download]  10.0% of   4.20MiB at  1.00MiB/s ETA 00:03",
		"[download]  50.0% of   4.20MiB at  2.00MiB/s ETA 00:01",
		"[download] 100.0% of   4.20MiB at  3.00MiB/s ETA 00:00",
		"",
	}, "\n")

	if _, err := pw.Write([]byte(input)); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := int(bar.State().CurrentPercent * 100)
	if got < 99 {
		t.Errorf("expected ~100, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./downloader/... -v -run TestProgressWriter
```

Expected: compile error — `progressWriter` not defined.

- [ ] **Step 3: Create `downloader/progress.go`**

```go
package downloader

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/iamNoah1/audiotap/internal/manager"
	"github.com/schollz/progressbar/v3"
)

var progressRe = regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`)

// progressWriter is an io.Writer that parses yt-dlp progress output
// and advances a progress bar.
type progressWriter struct {
	bar *progressbar.ProgressBar
	buf bytes.Buffer
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.buf.Write(p)
	data := pw.buf.Bytes()
	start := 0
	for i, b := range data {
		if b == '\n' || b == '\r' {
			line := string(data[start:i])
			if m := progressRe.FindStringSubmatch(line); m != nil {
				pct, _ := strconv.ParseFloat(m[1], 64)
				_ = pw.bar.Set(int(pct))
			}
			start = i + 1
		}
	}
	pw.buf.Reset()
	if start < len(data) {
		pw.buf.Write(data[start:])
	}
	return len(p), nil
}

// DownloadWithProgress is like Download but shows a live progress bar on stderr.
func DownloadWithProgress(rawURL string, cfg Config) (string, error) {
	if err := validateURL(rawURL); err != nil {
		return "", err
	}
	if err := checkDependencies(cfg.Format); err != nil {
		return "", err
	}

	outDir := cfg.OutputDir
	if outDir == "" {
		var err error
		outDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
	}

	outputTemplate := outDir + "/%(title)s.%(ext)s"

	args := []string{
		"--no-check-certificates",
		"-x",
		"--audio-format", cfg.Format,
		"--audio-quality", "0",
		"-o", outputTemplate,
		"--print", "after_move:filepath",
		"--newline", // Force \n in progress output for clean line parsing
		rawURL,
	}

	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	pw := &progressWriter{bar: bar}

	var stdout bytes.Buffer
	cmd := exec.Command(manager.BinaryPath(), args...)
	cmd.Stdout = &stdout
	cmd.Stderr = pw

	if err := cmd.Run(); err != nil {
		_ = bar.Finish()
		msg := strings.TrimSpace(pw.buf.String())
		if strings.Contains(msg, "network") || strings.Contains(msg, "Unable to download") {
			return "", fmt.Errorf("%w\nTip: make sure the URL is quoted to avoid shell escaping issues", ErrDownloadFailed)
		}
		return "", fmt.Errorf("%w\n%s", ErrDownloadFailed, msg)
	}

	_ = bar.Finish()
	outFile := strings.TrimSpace(stdout.String())
	return outFile, nil
}
```

- [ ] **Step 4: Run tests — expect them to pass**

```bash
go test ./downloader/... -v -run TestProgressWriter
```

Expected: all PASS.

- [ ] **Step 5: Build everything**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 6: Commit**

```bash
git add downloader/progress.go downloader/progress_test.go
git commit -m "feat(downloader): add progressWriter and DownloadWithProgress"
```

---

## Task 5: Wire into `cmd/root.go`

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Read the current file**

Read `cmd/root.go` in full before editing.

- [ ] **Step 2: Update `cmd/root.go`**

Replace the full contents of `cmd/root.go` with:

```go
package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/iamNoah1/audiotap/downloader"
	"github.com/iamNoah1/audiotap/internal/manager"
	"github.com/spf13/cobra"
)

var version = "dev"

var (
	outputDir string
	format    string
	inputFile string
	workers   int
)

var rootCmd = &cobra.Command{
	Use:     "audiotap [youtube-url ...]",
	Version: version,
	Short:   "Extract audio from YouTube videos",
	Long: `audiotap downloads audio from one or more YouTube URLs using yt-dlp.

Supported formats: mp3, opus, wav

Examples:
  audiotap "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
  audiotap "https://youtu.be/abc" "https://youtu.be/xyz" --format opus
  audiotap --input urls.txt --output-dir ./audio --workers 4`,
	Args:              cobra.ArbitraryArgs,
	PersistentPreRunE: setup,
	RunE:              run,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to save audio files (default: current directory)")
	rootCmd.Flags().StringVar(&format, "format", "opus", "Audio format: mp3, opus, wav")
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "File containing YouTube URLs, one per line")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "Number of parallel downloads")
}

// setup runs before any command and ensures yt-dlp is available.
func setup(_ *cobra.Command, _ []string) error {
	return manager.Ensure()
}

func run(_ *cobra.Command, args []string) error {
	validFormats := map[string]bool{"mp3": true, "opus": true, "wav": true}
	if !validFormats[format] {
		return fmt.Errorf("unsupported format %q — choose mp3, opus, or wav", format)
	}

	urls, err := collectURLs(args, inputFile)
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided — pass URLs as arguments or use --input <file>")
	}

	cfg := downloader.Config{
		OutputDir: outputDir,
		Format:    format,
	}

	if len(urls) == 1 {
		outFile, err := downloader.DownloadWithProgress(urls[0], cfg)
		if err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
		fmt.Printf("saved: %s\n", outFile)
		return nil
	}

	log.Printf("downloading %d URL(s) with %d worker(s)", len(urls), workers)
	start := time.Now()
	summary := downloader.RunBatch(urls, cfg, workers)
	summary.TotalWall = time.Since(start)

	printSummary(summary)

	if summary.Failed > 0 {
		os.Exit(1)
	}
	return nil
}

func collectURLs(args []string, inputFile string) ([]string, error) {
	var urls []string

	urls = append(urls, args...)

	if inputFile != "" {
		f, err := os.Open(inputFile)
		if err != nil {
			return nil, fmt.Errorf("opening input file: %w", err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			urls = append(urls, line)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading input file: %w", err)
		}
	}

	return urls, nil
}

func printSummary(s downloader.Summary) {
	const bar = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	fmt.Printf("\n%s\n", bar)
	fmt.Printf("  AudioTap — Done\n")
	fmt.Printf("%s\n", bar)
	fmt.Printf("  URLs processed : %d\n", s.Total)
	fmt.Printf("  Succeeded      : %d\n", s.Succeeded)
	fmt.Printf("  Failed         : %d\n", s.Failed)
	fmt.Printf("  Total time     : %s\n", s.TotalWall.Round(time.Second))
	fmt.Printf("%s\n", bar)
}
```

- [ ] **Step 3: Build and run all tests**

```bash
go build ./...
go test ./... -v
```

Expected: all PASS. Binary builds successfully.

- [ ] **Step 4: Verify help output shows opus as default**

```bash
./audiotap --help
```

Expected: `--format` flag shows `(default "opus")`.

- [ ] **Step 5: Commit**

```bash
git add cmd/root.go
git commit -m "feat(cmd): call manager.Ensure() on startup; default format opus; progress bar for single URL"
```

---

## Task 6: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read current README**

Read `README.md` in full before editing.

- [ ] **Step 2: Update Requirements section**

Replace the Requirements table and install snippet with:

```markdown
## Requirements

| Requirement | Notes |
|-------------|-------|
| Go 1.22+ | Only needed for `go install` / building from source |
| `ffmpeg` on `$PATH` | Required only for `--format mp3` or `--format wav` |

`yt-dlp` is downloaded automatically on first run — no manual installation needed.

To install ffmpeg (only if you need mp3/wav output):

\`\`\`bash
# macOS
brew install ffmpeg

# Ubuntu / Debian
sudo apt install ffmpeg
\`\`\`
```

- [ ] **Step 3: Update default format in Usage and Flags sections**

In the Flags table, change the `--format` default from `` `mp3` `` to `` `opus` ``.

In the Features section, update the format choice bullet to note that opus is the default and requires no extra tools.

- [ ] **Step 4: Add a note about auto-install to the Installation section**

After the `go install` code block, add:

```markdown
> **First run:** audiotap automatically downloads the `yt-dlp` binary for your platform (~10 MB) to `~/.local/share/audiotap/bin/`. This happens once and is silent on all subsequent runs.
```

- [ ] **Step 5: Build and check**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 6: Commit**

```bash
git add README.md
git commit -m "docs: update README for auto-install yt-dlp and opus default format"
```

---

## Self-Review

**Spec coverage:**
- ✅ Auto-install yt-dlp: Task 1 + 2
- ✅ Default format → opus: Task 5 (`init()`) + Task 6 (README)
- ✅ Clear error for ffmpeg when mp3/wav used: preserved in `checkDependencies` (Task 3)
- ✅ Progress bar for single download: Task 4 (`progressWriter`) + Task 5 (wired in `run`)
- ✅ Batch mode unchanged: `RunBatch` untouched
- ✅ One-line install message on stderr: Task 2 (`Ensure()`)
- ✅ Cache location documented: Task 2 + README Task 6

**Placeholder scan:** No TBDs, no "similar to Task N" references. All code blocks are complete.

**Type consistency:**
- `progressWriter.bar` is `*progressbar.ProgressBar` — used consistently in Task 4 tests and implementation
- `manager.BinaryPath()` returns `string` — used in `downloader.go` (Task 3) and `progress.go` (Task 4) as first arg to `exec.Command`
- `manager.Ensure()` returns `error` — handled in `PersistentPreRunE` in Task 5
- `downloader.DownloadWithProgress(rawURL string, cfg Config) (string, error)` — signature used in Task 5
