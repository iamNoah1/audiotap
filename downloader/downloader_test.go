package downloader

import (
	"testing"
)

func TestBuildArgs_NoCookies(t *testing.T) {
	cfg := Config{Format: "mp3", Cookies: ""}
	args := buildArgs("https://youtu.be/abc", "/out/%(title)s.%(ext)s", cfg)

	for i, a := range args {
		if a == "--cookies" {
			t.Errorf("unexpected --cookies flag at position %d", i)
		}
	}
	if args[len(args)-1] != "https://youtu.be/abc" {
		t.Errorf("URL must be the last arg, got %q", args[len(args)-1])
	}
}

func TestBuildArgs_WithCookies(t *testing.T) {
	cfg := Config{Format: "opus", Cookies: "/app/data/cookies.txt"}
	args := buildArgs("https://youtu.be/xyz", "/out/%(title)s.%(ext)s", cfg)

	var cookiesIdx int
	found := false
	for i, a := range args {
		if a == "--cookies" {
			found = true
			cookiesIdx = i
			break
		}
	}
	if !found {
		t.Fatal("expected --cookies flag but not found")
	}
	if cookiesIdx+1 >= len(args) {
		t.Fatal("--cookies flag has no value")
	}
	if args[cookiesIdx+1] != "/app/data/cookies.txt" {
		t.Errorf("expected cookies path %q, got %q", "/app/data/cookies.txt", args[cookiesIdx+1])
	}
	if args[len(args)-1] != "https://youtu.be/xyz" {
		t.Errorf("URL must be the last arg, got %q", args[len(args)-1])
	}
}

func TestBuildArgs_AudioFormat(t *testing.T) {
	for _, format := range []string{"mp3", "opus", "wav"} {
		cfg := Config{Format: format}
		args := buildArgs("https://youtu.be/abc", "/out/%(title)s.%(ext)s", cfg)
		found := false
		for i, a := range args {
			if a == "--audio-format" && i+1 < len(args) && args[i+1] == format {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("format %q not found in args: %v", format, args)
		}
	}
}
