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
			return err
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
