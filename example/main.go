// Command example walks a directory tree, compresses each text source file
// with boardgame, and reports per-extension and overall compression ratios,
// average compression time, and ASCII histograms of time and ratio vs size.
//
// Usage:
//
//	go run ./example /path/to/project
//	go run ./example -exclude vendor /path/to/project
//	go run ./example -include-vendored -workers 8 /path/to/project
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"boardgame"
)

// isText reports whether data looks like a text file by scanning for
// null bytes and checking that all bytes are valid ASCII text characters
// (printable ASCII, tabs, newlines, carriage returns).
func isText(data []byte) bool {
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}
	if bytes.ContainsRune(check, 0) {
		return false
	}
	for _, b := range check {
		if b > 0x7E && b != 0x0D {
			return false
		}
	}
	return true
}

type fileJob struct {
	path string
	ext  string
	src  []byte
}

type fileResult struct {
	ext      string
	origLen  int
	compLen  int
	ratio    float64
	duration time.Duration
}

type extStats struct {
	files       int
	origTotal   int
	compTotal   int
	ratioSum    float64
	durationSum time.Duration
}

func (s *extStats) avgSize() float64  { return float64(s.origTotal) / float64(s.files) }
func (s *extStats) avgRatio() float64 { return s.ratioSum / float64(s.files) }
func (s *extStats) avgTime() time.Duration {
	return s.durationSum / time.Duration(s.files)
}

func main() {
	exclude := flag.String("exclude", "", "additional directory name to exclude")
	includeVendored := flag.Bool("include-vendored", false, "include node_modules and vendor directories")
	maxSize := flag.Int("max-size", 20*1024, "maximum file size in bytes to consider (0 = unlimited)")
	workers := flag.Int("workers", runtime.NumCPU(), "number of parallel compression workers")
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./example [-exclude dir] [-include-vendored] [-max-size bytes] [-workers N] <directory>")
		os.Exit(1)
	}

	root, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	selfDir, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	exampleDir := filepath.Join(selfDir, "example")

	jobs := make(chan fileJob, *workers*2)
	results := make(chan fileResult, *workers*2)

	// Workers: compress files in parallel.
	var wg sync.WaitGroup
	for range *workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				start := time.Now()
				compressed, err := boardgame.Encode(job.src)
				elapsed := time.Since(start)
				if err != nil {
					continue
				}
				origLen, compLen, ratio := boardgame.Stats(job.src, compressed)
				results <- fileResult{
					ext:      job.ext,
					origLen:  origLen,
					compLen:  compLen,
					ratio:    ratio,
					duration: elapsed,
				}
			}
		}()
	}

	// Close results channel once all workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Walk the directory tree and send jobs.
	go func() {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" {
					return fs.SkipDir
				}
				if !*includeVendored && (name == "node_modules" || name == "vendor") {
					return fs.SkipDir
				}
				if *exclude != "" && name == *exclude {
					return fs.SkipDir
				}
				abs, _ := filepath.Abs(path)
				if abs == exampleDir {
					return fs.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext == "" {
				return nil
			}

			src, err := os.ReadFile(path)
			if err != nil || len(src) == 0 || !isText(src) {
				return nil
			}
			if *maxSize > 0 && len(src) > *maxSize {
				return nil
			}

			jobs <- fileJob{path: path, ext: ext, src: src}
			return nil
		})
		close(jobs)
	}()

	// Collect results.
	byExt := make(map[string]*extStats)
	var totalFiles int
	var totalOrig, totalComp int
	var totalRatioSum float64
	var totalDuration time.Duration

	for r := range results {
		st := byExt[r.ext]
		if st == nil {
			st = &extStats{}
			byExt[r.ext] = st
		}
		st.files++
		st.origTotal += r.origLen
		st.compTotal += r.compLen
		st.ratioSum += r.ratio
		st.durationSum += r.duration

		totalFiles++
		totalOrig += r.origLen
		totalComp += r.compLen
		totalRatioSum += r.ratio
		totalDuration += r.duration
	}

	if totalFiles == 0 {
		fmt.Println("no compressible files found")
		return
	}

	// Build sorted extension list (alphabetical for the table).
	exts := make([]string, 0, len(byExt))
	for ext := range byExt {
		exts = append(exts, ext)
	}
	sort.Strings(exts)

	// --- Summary table ---
	fmt.Printf("%-16s %6s %12s %12s %10s %12s\n", "Extension", "Files", "Original", "Compressed", "Avg Ratio", "Avg Time")
	fmt.Println(strings.Repeat("-", 72))
	for _, ext := range exts {
		st := byExt[ext]
		fmt.Printf("%-16s %6d %12d %12d %9.1f%% %12s\n",
			ext, st.files, st.origTotal, st.compTotal,
			st.avgRatio()*100, fmtDuration(st.avgTime()))
	}
	fmt.Println(strings.Repeat("-", 72))
	avgRatio := totalRatioSum / float64(totalFiles)
	avgTime := totalDuration / time.Duration(totalFiles)
	fmt.Printf("%-16s %6d %12d %12d %9.1f%% %12s\n",
		"TOTAL", totalFiles, totalOrig, totalComp,
		avgRatio*100, fmtDuration(avgTime))

	// --- Histograms (sorted by avg file size) ---
	bySizeExts := make([]string, len(exts))
	copy(bySizeExts, exts)
	sort.Slice(bySizeExts, func(i, j int) bool {
		return byExt[bySizeExts[i]].avgSize() < byExt[bySizeExts[j]].avgSize()
	})

	fmt.Println()
	printTimeHistogram(bySizeExts, byExt)
	fmt.Println()
	printRatioHistogram(bySizeExts, byExt)
}

const barWidth = 40

func printTimeHistogram(exts []string, byExt map[string]*extStats) {
	fmt.Println("Avg Compression Time vs Avg File Size (sorted by size)")
	fmt.Println(strings.Repeat("=", 80))

	// Find max time for scaling (use log scale since times span us to seconds).
	var maxLog float64
	for _, ext := range exts {
		us := float64(byExt[ext].avgTime().Microseconds())
		if us < 1 {
			us = 1
		}
		l := math.Log10(us)
		if l > maxLog {
			maxLog = l
		}
	}
	if maxLog < 1 {
		maxLog = 1
	}

	for _, ext := range exts {
		st := byExt[ext]
		us := float64(st.avgTime().Microseconds())
		if us < 1 {
			us = 1
		}
		barLen := int(math.Log10(us) / maxLog * barWidth)
		if barLen < 1 {
			barLen = 1
		}
		bar := strings.Repeat("#", barLen)
		fmt.Printf("%-14s %8s | %-*s %s\n",
			ext, fmtSize(int(st.avgSize())),
			barWidth, bar, fmtDuration(st.avgTime()))
	}
}

func printRatioHistogram(exts []string, byExt map[string]*extStats) {
	fmt.Println("Avg Compression Ratio vs Avg File Size (sorted by size)")
	fmt.Println(strings.Repeat("=", 80))

	for _, ext := range exts {
		st := byExt[ext]
		ratio := st.avgRatio()
		barLen := int(ratio * barWidth)
		if barLen < 0 {
			barLen = 0
		}
		if barLen > barWidth {
			barLen = barWidth
		}
		bar := strings.Repeat("#", barLen)
		fmt.Printf("%-14s %8s | %-*s %.1f%%\n",
			ext, fmtSize(int(st.avgSize())),
			barWidth, bar, ratio*100)
	}
}

func fmtDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%.0fus", float64(d.Microseconds()))
	case d < time.Second:
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func fmtSize(b int) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%dB", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
}
