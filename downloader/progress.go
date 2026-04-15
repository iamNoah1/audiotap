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
