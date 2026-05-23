package downloader

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iamNoah1/audiotap/internal/manager"
	"github.com/schollz/progressbar/v3"
)

var progressRe = regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`)

// progressWriter is an io.Writer that parses yt-dlp progress output,
// records the latest percentage for the ticker, and accumulates stderr
// lines for error reporting.
type progressWriter struct {
	bar         *progressbar.ProgressBar
	buf         bytes.Buffer
	stderrLines strings.Builder
	mu          sync.Mutex
	lastPct     float64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.buf.Write(p)
	data := pw.buf.Bytes()
	start := 0
	for i, b := range data {
		if b == '\n' || b == '\r' {
			line := string(data[start:i])
			pw.stderrLines.WriteString(line)
			pw.stderrLines.WriteByte('\n')
			if m := progressRe.FindStringSubmatch(line); m != nil {
				pct, _ := strconv.ParseFloat(m[1], 64)
				pw.mu.Lock()
				pw.lastPct = pct
				pw.mu.Unlock()
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

// DownloadWithProgress runs the download in a background goroutine and drives
// a live progress bar via a 100ms ticker — matching whisperbatch's animation style.
// Real yt-dlp percentage values are used when available; a hyperbolic time curve
// provides smooth animation as fallback.
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

	base := buildArgs(rawURL, outputTemplate, cfg)
	// buildArgs appends rawURL as the last element; insert --newline before it
	// so yt-dlp receives flags before the positional URL argument.
	args := make([]string, 0, len(base)+1)
	args = append(args, base[:len(base)-1]...)
	args = append(args, "--newline")
	args = append(args, rawURL)

	bar := progressbar.NewOptions(100,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	pw := &progressWriter{bar: bar}

	type dlResult struct {
		outFile string
		err     error
	}

	done := make(chan dlResult, 1)
	go func() {
		var stdout bytes.Buffer
		cmd := exec.Command(manager.BinaryPath(), args...)
		cmd.Stdout = &stdout
		cmd.Stderr = pw
		err := cmd.Run()
		done <- dlResult{outFile: strings.TrimSpace(stdout.String()), err: err}
	}()

	const spinFrames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	frames := []rune(spinFrames)
	var frame int
	start := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var res dlResult
loop:
	for {
		select {
		case res = <-done:
			break loop
		case <-ticker.C:
			elapsed := time.Since(start)

			pw.mu.Lock()
			pct := pw.lastPct
			pw.mu.Unlock()

			var target int
			if pct > 0 {
				target = int(pct)
			} else {
				// Hyperbolic curve: t/(t+k) → 1 asymptotically, k=30s, capped at 95.
				k := 30.0
				target = int(elapsed.Seconds() / (elapsed.Seconds() + k) * 95)
			}
			_ = bar.Set(target)

			spin := string(frames[frame%len(frames)])
			frame++
			bar.Describe(fmt.Sprintf("%s Downloading  %4s", spin, elapsed.Round(time.Second)))
		}
	}

	_ = bar.Set(100)
	_ = bar.Finish()

	if res.err != nil {
		msg := strings.TrimSpace(pw.stderrLines.String())
		if strings.Contains(msg, "network") || strings.Contains(msg, "Unable to download") {
			return "", fmt.Errorf("%w\nTip: make sure the URL is quoted to avoid shell escaping issues", ErrDownloadFailed)
		}
		return "", fmt.Errorf("%w\n%s", ErrDownloadFailed, msg)
	}

	return res.outFile, nil
}
