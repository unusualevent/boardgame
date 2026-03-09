// Command compare walks a directory tree, compresses each text source file
// with boardgame and several standard compression algorithms, and reports
// side-by-side compression ratios and timing.
//
// Usage:
//
//	go run ./example/compare /path/to/project
//	go run ./example/compare -exclude vendor /path/to/project
//	go run ./example/compare -include-vendored -max-size 0 -workers 8 /path/to/project
package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"git.risottobias.org/claude/boardgame"
	"github.com/andybalholm/brotli"
	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// algorithm defines a compression algorithm with name and compress function.
type algorithm struct {
	name     string
	compress func([]byte) ([]byte, error)
}

func gzipCompress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func snappyCompress(src []byte) ([]byte, error) {
	return snappy.Encode(nil, src), nil
}

func zstdCompress(src []byte) ([]byte, error) {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, err
	}
	defer enc.Close()
	return enc.EncodeAll(src, nil), nil
}

func lz4Compress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func brotliCompress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := brotli.NewWriterLevel(&buf, brotli.DefaultCompression)
	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func boardgameCompress(src []byte) ([]byte, error) {
	return boardgame.Encode(src)
}

var algorithms = []algorithm{
	{"boardgame", boardgameCompress},
	{"gzip", gzipCompress},
	{"snappy", snappyCompress},
	{"zstd", zstdCompress},
	{"lz4", lz4Compress},
	{"brotli", brotliCompress},
}

// isText reports whether data looks like a text file (no null bytes in first 8KB).
func isText(data []byte) bool {
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}
	return !bytes.ContainsRune(check, 0)
}

type fileJob struct {
	path string
	ext  string
	src  []byte
}

// algoResult holds compression results for one algorithm on one file.
type algoResult struct {
	compLen  int
	ratio    float64
	duration time.Duration
}

type fileResult struct {
	ext     string
	origLen int
	algos   []algoResult // one per algorithm, same order as algorithms slice
}

type algoStats struct {
	compTotal   int
	ratioSum    float64
	durationSum time.Duration
}

type extStats struct {
	files     int
	origTotal int
	algos     []algoStats // one per algorithm
}

func (s *extStats) avgSize() float64 { return float64(s.origTotal) / float64(s.files) }

func main() {
	exclude := flag.String("exclude", "", "additional directory name to exclude")
	includeVendored := flag.Bool("include-vendored", false, "include node_modules and vendor directories")
	maxSize := flag.Int("max-size", 20*1024, "maximum file size in bytes (0 = unlimited)")
	workers := flag.Int("workers", runtime.NumCPU(), "number of parallel compression workers")
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./example/compare [-exclude dir] [-include-vendored] [-max-size bytes] [-workers N] <directory>")
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

	// Workers: compress files with all algorithms in parallel.
	var wg sync.WaitGroup
	for range *workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				r := fileResult{
					ext:     job.ext,
					origLen: len(job.src),
					algos:   make([]algoResult, len(algorithms)),
				}
				for i, alg := range algorithms {
					start := time.Now()
					compressed, err := alg.compress(job.src)
					elapsed := time.Since(start)
					if err != nil {
						r.algos[i] = algoResult{compLen: len(job.src), ratio: 0, duration: elapsed}
						continue
					}
					compLen := len(compressed)
					ratio := 1.0 - float64(compLen)/float64(len(job.src))
					r.algos[i] = algoResult{compLen: compLen, ratio: ratio, duration: elapsed}
				}
				results <- r
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Walk directory tree and send jobs.
	go func() {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
	var totalOrig int
	totalAlgos := make([]algoStats, len(algorithms))

	for r := range results {
		st := byExt[r.ext]
		if st == nil {
			st = &extStats{algos: make([]algoStats, len(algorithms))}
			byExt[r.ext] = st
		}
		st.files++
		st.origTotal += r.origLen
		for i, ar := range r.algos {
			st.algos[i].compTotal += ar.compLen
			st.algos[i].ratioSum += ar.ratio
			st.algos[i].durationSum += ar.duration
		}

		totalFiles++
		totalOrig += r.origLen
		for i, ar := range r.algos {
			totalAlgos[i].compTotal += ar.compLen
			totalAlgos[i].ratioSum += ar.ratio
			totalAlgos[i].durationSum += ar.duration
		}
	}

	if totalFiles == 0 {
		fmt.Println("no compressible files found")
		return
	}

	// Sorted extension list.
	exts := make([]string, 0, len(byExt))
	for ext := range byExt {
		exts = append(exts, ext)
	}
	sort.Strings(exts)

	// --- Print comparison tables ---
	printRatioTable(exts, byExt, totalFiles, totalOrig, totalAlgos)
	fmt.Println()
	printTimeTable(exts, byExt, totalFiles, totalAlgos)
	fmt.Println()
	printOverallSummary(totalFiles, totalOrig, totalAlgos)
}

// printRatioTable prints per-extension average compression ratio for each algorithm.
func printRatioTable(exts []string, byExt map[string]*extStats, totalFiles, totalOrig int, totalAlgos []algoStats) {
	fmt.Println("Avg Compression Ratio by Extension (%)")
	fmt.Println(strings.Repeat("=", 100))

	// Header.
	fmt.Printf("%-12s %5s %8s", "Extension", "Files", "AvgSize")
	for _, alg := range algorithms {
		fmt.Printf(" %10s", alg.name)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 100))

	for _, ext := range exts {
		st := byExt[ext]
		fmt.Printf("%-12s %5d %8s", ext, st.files, fmtSize(int(st.avgSize())))
		for i := range algorithms {
			avgRatio := st.algos[i].ratioSum / float64(st.files) * 100
			fmt.Printf(" %9.1f%%", avgRatio)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-12s %5d %8s", "TOTAL", totalFiles, fmtSize(totalOrig/totalFiles))
	for i := range algorithms {
		avgRatio := totalAlgos[i].ratioSum / float64(totalFiles) * 100
		fmt.Printf(" %9.1f%%", avgRatio)
	}
	fmt.Println()
}

// printTimeTable prints per-extension average compression time for each algorithm.
func printTimeTable(exts []string, byExt map[string]*extStats, totalFiles int, totalAlgos []algoStats) {
	fmt.Println("Avg Compression Time by Extension")
	fmt.Println(strings.Repeat("=", 100))

	fmt.Printf("%-12s %5s %8s", "Extension", "Files", "AvgSize")
	for _, alg := range algorithms {
		fmt.Printf(" %10s", alg.name)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 100))

	for _, ext := range exts {
		st := byExt[ext]
		fmt.Printf("%-12s %5d %8s", ext, st.files, fmtSize(int(st.avgSize())))
		for i := range algorithms {
			avgTime := st.algos[i].durationSum / time.Duration(st.files)
			fmt.Printf(" %10s", fmtDuration(avgTime))
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-12s %5d %8s", "TOTAL", totalFiles, "")
	for i := range algorithms {
		avgTime := totalAlgos[i].durationSum / time.Duration(totalFiles)
		fmt.Printf(" %10s", fmtDuration(avgTime))
	}
	fmt.Println()
}

// printOverallSummary prints a compact summary comparing all algorithms.
func printOverallSummary(totalFiles, totalOrig int, totalAlgos []algoStats) {
	fmt.Println("Overall Summary")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Files: %d    Original size: %s\n\n", totalFiles, fmtSize(totalOrig))
	fmt.Printf("%-12s %10s %10s %12s %12s\n", "Algorithm", "Avg Ratio", "Total Out", "Avg Time", "Throughput")
	fmt.Println(strings.Repeat("-", 60))

	for i, alg := range algorithms {
		avgRatio := totalAlgos[i].ratioSum / float64(totalFiles) * 100
		avgTime := totalAlgos[i].durationSum / time.Duration(totalFiles)
		avgSize := float64(totalOrig) / float64(totalFiles)
		var throughput string
		if avgTime > 0 {
			mbps := avgSize / avgTime.Seconds() / (1024 * 1024)
			if mbps >= 1 {
				throughput = fmt.Sprintf("%.1f MB/s", mbps)
			} else {
				throughput = fmt.Sprintf("%.0f KB/s", avgSize/avgTime.Seconds()/1024)
			}
		}
		fmt.Printf("%-12s %9.1f%% %10s %12s %12s\n",
			alg.name, avgRatio, fmtSize(totalAlgos[i].compTotal), fmtDuration(avgTime), throughput)
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
