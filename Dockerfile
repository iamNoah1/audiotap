# ─── Stage 1: build the Go binary ─────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache dependency downloads separately from source code.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
        -ldflags="-s -w" \
        -o audiotap .

# ─── Stage 2: runtime with yt-dlp + ffmpeg ─────────────────────────────────────
FROM python:3.11-slim

LABEL org.opencontainers.image.title="audiotap" \
      org.opencontainers.image.description="Extract audio from YouTube videos" \
      org.opencontainers.image.source="https://github.com/iamNoah1/audiotap" \
      org.opencontainers.image.licenses="MIT"

# ffmpeg is required for mp3/wav re-encoding.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg \
 && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir yt-dlp

COPY --from=builder /app/audiotap /usr/local/bin/audiotap

# Mount a directory at /output for downloaded files.
# Example:
#   docker run --rm -v /host/audio:/output \
#     ghcr.io/iamnoah1/audiotap "https://youtu.be/..." --output-dir /output
ENTRYPOINT ["audiotap"]
