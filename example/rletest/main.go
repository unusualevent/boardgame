// Command rletest measures the impact of RLE preprocessing on boardgame
// encoding time and compression ratio. It compresses each file twice:
// once normally, once with whitespace runs (>= threshold) collapsed
// before encoding. This isolates the time/ratio tradeoff without
// changing the wire format.
package main

import (
	"bytes"
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
)

func isText(data []byte) bool {
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}
	return !bytes.ContainsRune(check, 0)
}

// collapseRuns replaces runs of spaces or tabs (length >= threshold)
// with exactly 2 characters of the same type. This simulates what
// RLE would do to the SA input size without implementing a real
// RLE encoding.
func collapseRuns(src []byte, threshold int) []byte {
	out := make([]byte, 0, len(src))
	i := 0
	for i < len(src) {
		if src[i] == ' ' || src[i] == '\t' {
			ch := src[i]
			start := i
			for i < len(src) && src[i] == ch {
				i++
			}
			rl := i - start
			if rl >= threshold {
				out = append(out, ch, ch)
			} else {
				out = append(out, src[start:i]...)
			}
		} else {
			out = append(out, src[i])
			i++
		}
	}
	return out
}

type fileJob struct {
	ext string
	src []byte
}

type fileResult struct {
	ext          string
	origLen      int
	normalComp   int
	normalTime   time.Duration
	normalRatio  float64
	rleComp      int
	rleTime      time.Duration
	rleRatio     float64
	collapsedLen int
}

type extStats struct {
	files            int
	origTotal        int
	normalCompTotal  int
	normalRatioSum   float64
	normalTimeSum    time.Duration
	rleCompTotal     int
	rleRatioSum      float64
	rleTimeSum       time.Duration
	collapsedLenSum  int
}

func (s *extStats) avgSize() float64 { return float64(s.origTotal) / float64(s.files) }

func main() {
	maxSize := flag.Int("max-size", 20*1024, "maximum file size (0 = unlimited)")
	workers := flag.Int("workers", runtime.NumCPU(), "parallel workers")
	threshold := flag.Int("threshold", 4, "minimum run length to collapse")
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./example/rletest [-threshold N] <directory>")
		os.Exit(1)
	}
	root, _ = filepath.Abs(root)

	jobs := make(chan fileJob, *workers*2)
	results := make(chan fileResult, *workers*2)

	var wg sync.WaitGroup
	for range *workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Normal encode.
				t0 := time.Now()
				normalComp, err := boardgame.Encode(job.src)
				normalTime := time.Since(t0)
				if err != nil {
					continue
				}

				// RLE-preprocessed encode.
				collapsed := collapseRuns(job.src, *threshold)
				t1 := time.Now()
				rleComp, err := boardgame.Encode(collapsed)
				rleTime := time.Since(t1)
				if err != nil {
					continue
				}

				_, normalCompLen, normalRatio := boardgame.Stats(job.src, normalComp)
				// For RLE ratio, measure against original size (not collapsed).
				rleRatio := 1.0 - float64(len(rleComp))/float64(len(job.src))

				results <- fileResult{
					ext:          job.ext,
					origLen:      len(job.src),
					normalComp:   normalCompLen,
					normalTime:   normalTime,
					normalRatio:  normalRatio,
					rleComp:      len(rleComp),
					rleTime:      rleTime,
					rleRatio:     rleRatio,
					collapsedLen: len(collapsed),
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == "vendor" {
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
			jobs <- fileJob{ext: ext, src: src}
			return nil
		})
		close(jobs)
	}()

	byExt := make(map[string]*extStats)
	var totalFiles int
	var totalOrig int
	var totalNormalComp, totalRLEComp int
	var totalNormalRatio, totalRLERatio float64
	var totalNormalTime, totalRLETime time.Duration
	var totalCollapsed int

	for r := range results {
		st := byExt[r.ext]
		if st == nil {
			st = &extStats{}
			byExt[r.ext] = st
		}
		st.files++
		st.origTotal += r.origLen
		st.normalCompTotal += r.normalComp
		st.normalRatioSum += r.normalRatio
		st.normalTimeSum += r.normalTime
		st.rleCompTotal += r.rleComp
		st.rleRatioSum += r.rleRatio
		st.rleTimeSum += r.rleTime
		st.collapsedLenSum += r.collapsedLen

		totalFiles++
		totalOrig += r.origLen
		totalNormalComp += r.normalComp
		totalRLEComp += r.rleComp
		totalNormalRatio += r.normalRatio
		totalRLERatio += r.rleRatio
		totalNormalTime += r.normalTime
		totalRLETime += r.rleTime
		totalCollapsed += r.collapsedLen
	}

	if totalFiles == 0 {
		fmt.Println("no files found")
		return
	}

	exts := make([]string, 0, len(byExt))
	for ext := range byExt {
		exts = append(exts, ext)
	}
	sort.Slice(exts, func(i, j int) bool {
		return byExt[exts[i]].origTotal > byExt[exts[j]].origTotal
	})

	fmt.Printf("RLE preprocessing test (threshold >= %d)\n", *threshold)
	fmt.Println(strings.Repeat("=", 115))
	fmt.Printf("%-12s %5s %8s %10s %10s %10s %10s %9s %9s\n",
		"Extension", "Files", "AvgSize", "NormalTime", "RLE Time", "Speedup", "Shrink%", "NormRatio", "RLE Ratio")
	fmt.Println(strings.Repeat("-", 115))

	for _, ext := range exts {
		st := byExt[ext]
		normalAvg := st.normalTimeSum / time.Duration(st.files)
		rleAvg := st.rleTimeSum / time.Duration(st.files)
		speedup := float64(normalAvg) / float64(rleAvg)
		shrink := (1.0 - float64(st.collapsedLenSum)/float64(st.origTotal)) * 100
		normalRatio := st.normalRatioSum / float64(st.files) * 100
		rleRatio := st.rleRatioSum / float64(st.files) * 100
		fmt.Printf("%-12s %5d %8s %10s %10s %9.2fx %8.1f%% %8.1f%% %8.1f%%\n",
			ext, st.files, fmtSize(int(st.avgSize())),
			fmtDuration(normalAvg), fmtDuration(rleAvg), speedup,
			shrink, normalRatio, rleRatio)
	}

	fmt.Println(strings.Repeat("-", 115))
	normalAvg := totalNormalTime / time.Duration(totalFiles)
	rleAvg := totalRLETime / time.Duration(totalFiles)
	speedup := float64(normalAvg) / float64(rleAvg)
	shrink := (1.0 - float64(totalCollapsed)/float64(totalOrig)) * 100
	normalRatio := totalNormalRatio / float64(totalFiles) * 100
	rleRatio := totalRLERatio / float64(totalFiles) * 100
	fmt.Printf("%-12s %5d %8s %10s %10s %9.2fx %8.1f%% %8.1f%% %8.1f%%\n",
		"TOTAL", totalFiles, fmtSize(totalOrig/totalFiles),
		fmtDuration(normalAvg), fmtDuration(rleAvg), speedup,
		shrink, normalRatio, rleRatio)

	fmt.Println()
	fmt.Printf("Ratio change: %.1f%% → %.1f%% (%.1f%% %s)\n",
		normalRatio, rleRatio,
		abs(rleRatio-normalRatio),
		func() string {
			if rleRatio >= normalRatio {
				return "better"
			}
			return "worse"
		}())
	fmt.Printf("Time change: %s → %s (%.2fx)\n",
		fmtDuration(normalAvg), fmtDuration(rleAvg), speedup)
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
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
