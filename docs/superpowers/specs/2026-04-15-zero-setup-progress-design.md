# Design: Zero-setup install + single-download progress

**Date:** 2026-04-15  
**Status:** Approved

---

## Problem

1. `audiotap` requires users to manually install `yt-dlp` (Python) and `ffmpeg` before it works. This is friction that makes `go install` feel incomplete.
2. Single-URL downloads produce no output while running — users have no way to know if the tool is working or hung.

---

## Goals

- `go install github.com/iamNoah1/audiotap@latest` → run it → it works, no extra steps.
- A single download shows a live progress bar with percentage.
- Batch downloads are unchanged.

---

## Non-goals

- Auto-updating yt-dlp after installation.
- Eliminating ffmpeg (it remains required for mp3/wav; we just don't require it by default).
- Progress bar in batch mode per-download (would produce interleaved output).

---

## Design

### 1. yt-dlp auto-install (`internal/manager/`)

A new package `internal/manager` handles yt-dlp lifecycle.

**`Ensure() error`**  
Called once at startup. Resolution order:
1. Check `PATH` — if `yt-dlp` is found, record the path and return.
2. Check the cache dir — if the binary exists there, record the path and return.
3. Download the official yt-dlp binary for the current platform from GitHub releases into the cache dir, mark it executable, record the path, and return.

Cache dir:
- Linux/macOS: `~/.local/share/audiotap/bin/`
- Windows: `%APPDATA%\audiotap\bin\`

Download URL pattern (resolved at runtime to latest release):
```
https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_<platform>
```

Platform mapping:
| GOOS/GOARCH | yt-dlp artifact |
|---|---|
| linux/amd64 | `yt-dlp_linux` |
| linux/arm64 | `yt-dlp_linux_aarch64` |
| darwin/amd64 | `yt-dlp_macos_legacy` |
| darwin/arm64 | `yt-dlp_macos` |
| windows/amd64 | `yt-dlp.exe` |

Download shows a one-line status message to stderr:
```
audiotap: installing yt-dlp (first run)... done
```

**`BinaryPath() string`**  
Returns the resolved path after `Ensure()` has been called.

**Error handling:** If download fails (no network, unsupported platform), return a clear error with manual install instructions. The tool does not proceed.

---

### 2. Default format change

`--format` default changes from `mp3` → `opus`.

YouTube serves opus audio natively. yt-dlp can extract it without transcoding — no ffmpeg required. Users who want mp3 or wav pass `--format mp3` / `--format wav` explicitly; ffmpeg is still required for those and audiotap will say so clearly.

---

### 3. Single-download progress bar (`downloader/progress.go`)

yt-dlp writes progress to stderr:
```
[download]  23.4% of   4.20MiB at  1.23MiB/s ETA 00:02
```

A new `progressWriter` implements `io.Writer`. It:
- Buffers stderr line by line
- Matches lines against `\[download\]\s+(\d+(?:\.\d+)?)%`
- On match, calls `bar.Set(int(pct))` on a `schollz/progressbar` (0–100)
- Non-matching lines are discarded (yt-dlp prints a lot of noise)

`downloader.go` gains a `DownloadWithProgress(rawURL string, cfg Config, w io.Writer)` variant that wires `cmd.Stderr = progressWriter`. The original `Download()` function is unchanged (used by batch workers internally).

`cmd/root.go` calls `DownloadWithProgress` for single-URL downloads (when `len(urls) == 1`), passing `os.Stderr` as the writer so the bar renders in the terminal.

---

## File changes

| File | Change |
|---|---|
| `internal/manager/manager.go` | New — yt-dlp resolution + auto-download |
| `downloader/progress.go` | New — `progressWriter` and `DownloadWithProgress` |
| `downloader/downloader.go` | Use `manager.BinaryPath()` instead of hardcoded `"yt-dlp"`; remove `ErrYtDlpNotFound` |
| `cmd/root.go` | Change default format to `opus`; call `manager.Ensure()` at startup; use `DownloadWithProgress` for single URL |

---

## Testing

- `internal/manager`: unit test `Ensure()` with a mock HTTP server serving a fake binary; test cache-hit path (no network call on second run); test platform mapping.
- `downloader/progress.go`: unit test `progressWriter` — feed known yt-dlp output lines, assert bar advances correctly.
- `downloader/downloader.go`: existing tests unchanged.
- `cmd/root.go`: existing integration tests update expected default format to `opus`.
