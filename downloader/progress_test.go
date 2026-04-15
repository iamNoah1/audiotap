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
