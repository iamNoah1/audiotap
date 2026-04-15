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
