package downloader

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Result captures the outcome of downloading a single URL.
type Result struct {
	URL     string
	OutFile string
	Elapsed time.Duration
	Err     error
}

// Summary holds aggregate statistics for a completed batch.
type Summary struct {
	Total     int
	Succeeded int
	Failed    int
	TotalWall time.Duration
	Results   []Result
}

// RunBatch downloads all URLs using a fixed-size worker pool.
// Each in-flight download drives a 100ms ticker for smooth bar animation,
// matching whisperbatch's style. Failed downloads are printed as they occur.
func RunBatch(urls []string, cfg Config, workers int) Summary {
	jobs := make(chan string, len(urls))
	for _, u := range urls {
		jobs <- u
	}
	close(jobs)

	total := len(urls)
	barMax := total * 100

	var (
		mu               sync.Mutex
		results          = make([]Result, 0, total)
		completedElapsed []time.Duration
		wg               sync.WaitGroup
	)

	barStats := func() (int, time.Duration, string) {
		mu.Lock()
		n := len(completedElapsed)
		var sum time.Duration
		for _, d := range completedElapsed {
			sum += d
		}
		mu.Unlock()
		if n == 0 {
			return 0, 0, ""
		}
		avg := sum / time.Duration(n)
		remaining := total - n
		var eta string
		if remaining > 0 {
			est := avg * time.Duration(remaining)
			eta = fmt.Sprintf("~%s", est.Round(time.Second))
		}
		return n, avg, eta
	}

	fileFraction := func(elapsed, avg time.Duration) float64 {
		if avg > 0 {
			f := elapsed.Seconds() / avg.Seconds()
			if f > 0.95 {
				return 0.95
			}
			return f
		}
		k := 60.0
		return elapsed.Seconds() / (elapsed.Seconds() + k)
	}

	bar := progressbar.NewOptions(
		barMax,
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

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				name := urlLabel(u, 24)
				fileStart := time.Now()

				tickDone := make(chan struct{})
				go func() {
					const spinFrames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
					frames := []rune(spinFrames)
					var frame int
					ticker := time.NewTicker(100 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-tickDone:
							return
						case <-ticker.C:
							elapsed := time.Since(fileStart)

							// Read n dynamically so concurrent workers advance
							// through slots correctly as completions happen.
							mu.Lock()
							n := len(completedElapsed)
							mu.Unlock()

							_, avg, eta := barStats()
							frac := fileFraction(elapsed, avg)
							target := n*100 + int(frac*100)
							_ = bar.Set(target)

							spin := string(frames[frame%len(frames)])
							frame++
							fileNum := n + 1
							if fileNum > total {
								fileNum = total
							}
							var desc string
							if eta != "" {
								desc = fmt.Sprintf("%s [%d/%d] %-24s %4s  %s left", spin, fileNum, total, name, elapsed.Round(time.Second), eta)
							} else {
								desc = fmt.Sprintf("%s [%d/%d] %-24s %4s", spin, fileNum, total, name, elapsed.Round(time.Second))
							}
							bar.Describe(desc)
						}
					}
				}()

				r := downloadOne(u, cfg)
				close(tickDone)

				mu.Lock()
				completedElapsed = append(completedElapsed, r.Elapsed)
				results = append(results, r)
				n := len(completedElapsed)
				mu.Unlock()

				_ = bar.Set(n * 100)
				if r.Err != nil {
					fmt.Fprintf(os.Stderr, "\nFAILED %s: %v\n", u, r.Err)
				}
			}
		}()
	}

	wg.Wait()
	_ = bar.Finish()

	summary := Summary{Total: len(results), Results: results}
	for _, r := range results {
		if r.Err == nil {
			summary.Succeeded++
		} else {
			summary.Failed++
		}
	}
	return summary
}

func downloadOne(u string, cfg Config) Result {
	start := time.Now()
	outFile, err := Download(u, cfg)
	return Result{
		URL:     u,
		OutFile: outFile,
		Elapsed: time.Since(start),
		Err:     err,
	}
}

// urlLabel extracts the YouTube video ID for display, or truncates the URL.
func urlLabel(rawURL string, n int) string {
	if u, err := url.Parse(rawURL); err == nil {
		if id := u.Query().Get("v"); id != "" {
			return truncateName(id, n)
		}
		// youtu.be/ID format
		if strings.Contains(u.Host, "youtu.be") {
			id := strings.TrimPrefix(u.Path, "/")
			return truncateName(id, n)
		}
	}
	return truncateName(rawURL, n)
}

func truncateName(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
