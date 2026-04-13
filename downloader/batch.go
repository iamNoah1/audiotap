package downloader

import (
	"fmt"
	"os"
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
// Progress is displayed on stderr. Failed downloads are printed as they occur.
func RunBatch(urls []string, cfg Config, workers int) Summary {
	jobs := make(chan string, len(urls))
	for _, u := range urls {
		jobs <- u
	}
	close(jobs)

	var (
		mu      sync.Mutex
		results = make([]Result, 0, len(urls))
		wg      sync.WaitGroup
	)

	bar := progressbar.NewOptions(
		len(urls),
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
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
				r := downloadOne(u, cfg)
				_ = bar.Add(1)
				if r.Err != nil {
					fmt.Fprintf(os.Stderr, "\nFAILED %s: %v\n", u, r.Err)
				}
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	_ = bar.Finish()

	summary := Summary{
		Total:   len(results),
		Results: results,
	}
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
