# audiotap

[![CI](https://img.shields.io/github/actions/workflow/status/iamNoah1/audiotap/ci.yml?branch=main&label=CI)](https://github.com/iamNoah1/audiotap/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/iamNoah1/audiotap)](https://github.com/iamNoah1/audiotap/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/iamNoah1/audiotap)](https://goreportcard.com/report/github.com/iamNoah1/audiotap)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/iamNoah1/audiotap)](go.mod)

A fast CLI for extracting audio from YouTube videos — single URL or batch.  
Wraps [yt-dlp](https://github.com/yt-dlp/yt-dlp) with a clean interface, parallel downloads, and a progress bar.

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  AudioTap — Done
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  URLs processed : 12
  Succeeded      : 12
  Failed         : 0
  Total time     : 43s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## Features

- **Single or batch** — pass one URL, many URLs, or a file of URLs
- **Parallel downloads** — configurable worker pool, defaults to your CPU count
- **Format choice** — `mp3`, `opus`, or `wav`
- **Graceful failures** — one failed URL doesn't abort the batch
- **Progress bar** — live feedback with per-URL failure reporting
- **Chainable** — pairs naturally with [whisperbatch](https://github.com/iamNoah1/whisperbatch) for transcription

---

## Requirements

| Requirement | Notes |
|-------------|-------|
| Go 1.22+ | Only needed for `go install` / building from source |
| `yt-dlp` on `$PATH` | See install instructions below |
| `ffmpeg` on `$PATH` | Required for mp3 and wav conversion |

Install yt-dlp and ffmpeg:

```bash
# macOS
brew install ffmpeg yt-dlp

# Ubuntu / Debian
sudo apt install ffmpeg
pip install yt-dlp

# pip (any platform)
pip install yt-dlp
```

---

## Installation

### go install (recommended)

```bash
go install github.com/iamNoah1/audiotap@latest
```

### Pre-built binaries

Download the binary for your platform from the [Releases page](https://github.com/iamNoah1/audiotap/releases/latest), extract, and place on your `$PATH`.

```bash
# Example: Linux amd64
curl -L https://github.com/iamNoah1/audiotap/releases/latest/download/audiotap_linux_amd64.tar.gz \
  | tar -xz -C /usr/local/bin
```

### Docker

```bash
# Pull the image (includes yt-dlp and ffmpeg)
docker pull ghcr.io/iamnoah1/audiotap:latest

# Download a single video
docker run --rm \
  -v "$(pwd):/output" \
  ghcr.io/iamnoah1/audiotap "https://youtu.be/..." --output-dir /output

# Batch download from a URL list
docker run --rm \
  -v "$(pwd)/urls.txt:/urls.txt:ro" \
  -v "$(pwd)/audio:/output" \
  ghcr.io/iamnoah1/audiotap --input /urls.txt --output-dir /output
```

### Build from source

```bash
git clone https://github.com/iamNoah1/audiotap.git
cd audiotap
make build        # → ./audiotap
make install      # → $GOPATH/bin/audiotap
```

---

## Usage

```bash
# Single URL (mp3 by default)
audiotap "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# Save to a specific directory
audiotap "https://youtu.be/dQw4w9WgXcQ" --output-dir ./audio

# Different format
audiotap "https://youtu.be/dQw4w9WgXcQ" --format opus

# Multiple URLs in one command
audiotap "https://youtu.be/abc" "https://youtu.be/xyz" --output-dir ./audio

# Batch from a file (one URL per line, # comments supported)
audiotap --input urls.txt --output-dir ./audio --workers 4

# Pipe into whisperbatch for transcription
audiotap --input urls.txt --output-dir ./audio && whisperbatch -i ./audio
```

---

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output-dir` | | string | current dir | Directory to save audio files |
| `--format` | | string | `mp3` | Audio format: `mp3`, `opus`, `wav` |
| `--input` | `-i` | string | | File of YouTube URLs, one per line |
| `--workers` | `-w` | int | CPU count | Parallel download workers (batch mode) |
| `--version` | | | | Print version and exit |

---

## URL File Format

The `--input` file accepts one URL per line. Blank lines and lines starting with `#` are ignored:

```
# Tech talks
https://www.youtube.com/watch?v=abc123
https://www.youtube.com/watch?v=def456

# Music
https://youtu.be/xyz789
```

---

## Error Handling

| Error | Message |
|-------|---------|
| `yt-dlp` not installed | `yt-dlp not found. Run: pip install yt-dlp` |
| `ffmpeg` not installed | `ffmpeg not found. Install via brew install ffmpeg (macOS) or apt install ffmpeg (Linux)` |
| Invalid URL | `invalid YouTube URL` |
| Network / download failure | `download failed. Check your connection or try again` |

---

## Development

```bash
make build        # compile
make test         # run tests
make vet          # go vet
make lint         # golangci-lint (requires golangci-lint installed)
make release-dry  # goreleaser snapshot (requires goreleaser installed)
make clean        # remove artifacts
```

---

## Releasing

Push a semver tag — the release workflow handles everything:

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will:
1. Build binaries for Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)
2. Build and push multi-arch Docker images to GHCR
3. Create a GitHub Release with a changelog and `checksums.txt`

---

## Project Structure

```
audiotap/
├── main.go                   Entry point
├── cmd/root.go               Cobra command, flag definitions, summary output
├── downloader/
│   ├── downloader.go         yt-dlp wrapper, URL validation, dependency checks
│   └── batch.go              Worker pool orchestration
├── Dockerfile                Self-contained image (Go build + runtime)
├── Dockerfile.release        Runtime-only image (used by GoReleaser)
├── .goreleaser.yaml          Cross-platform release configuration
└── .github/workflows/        CI and release pipelines
```

---

## License

[MIT](LICENSE) © iamNoah1
