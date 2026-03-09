// Command whitespace measures what fraction of suffix array positions
// in source files come from whitespace runs (spaces and tabs). This
// helps determine whether RLE preprocessing would meaningfully shrink
// the SA and speed up encoding.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func isText(data []byte) bool {
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}
	return !bytes.ContainsRune(check, 0)
}

// isLiteral matches boardgame's definition: printable ASCII + tab + newline.
func isLiteral(b byte) bool {
	return (b >= 0x20 && b <= 0x7E) || b == 0x09 || b == 0x0A
}

type extStats struct {
	files           int
	totalPositions  int64
	spacePositions  int64
	tabPositions    int64
	spaceRunCount   int
	tabRunCount     int
	spaceRunLenSum  int
	tabRunLenSum    int
	totalBytes      int64
	totalWhitespace int64
}

func analyze(data []byte) (totalPos, spacePos, tabPos int64, spaceRuns, tabRuns, spaceRunLen, tabRunLen int, totalWS int64) {
	s := string(data)
	// Count literal-run positions and whitespace within them (same as boardgame's literalRuns + buildSA).
	i := 0
	for i < len(s) {
		if !isLiteral(s[i]) {
			i++
			continue
		}
		start := i
		for i < len(s) && isLiteral(s[i]) {
			i++
		}
		if i-start < 2 {
			continue
		}
		// This run [start, i) would be in the SA.
		for j := start; j < i; j++ {
			totalPos++
			switch s[j] {
			case ' ':
				spacePos++
				totalWS++
			case '\t':
				tabPos++
				totalWS++
			}
		}
	}

	// Count whitespace runs (consecutive spaces or tabs).
	for j := 0; j < len(s); {
		if s[j] == ' ' {
			runStart := j
			for j < len(s) && s[j] == ' ' {
				j++
			}
			rl := j - runStart
			if rl >= 2 {
				spaceRuns++
				spaceRunLen += rl
			}
		} else if s[j] == '\t' {
			runStart := j
			for j < len(s) && s[j] == '\t' {
				j++
			}
			rl := j - runStart
			if rl >= 2 {
				tabRuns++
				tabRunLen += rl
			}
		} else {
			j++
		}
	}
	return
}

func main() {
	maxSize := flag.Int("max-size", 20*1024, "maximum file size (0 = unlimited)")
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./example/whitespace <directory>")
		os.Exit(1)
	}

	root, _ = filepath.Abs(root)
	byExt := make(map[string]*extStats)

	var globalTotal, globalSpace, globalTab, globalWS, globalBytes int64
	var globalSpaceRuns, globalTabRuns, globalSpaceRunLen, globalTabRunLen int

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

		totalPos, spacePos, tabPos, spaceRuns, tabRuns, spaceRunLen, tabRunLen, totalWS := analyze(src)

		st := byExt[ext]
		if st == nil {
			st = &extStats{}
			byExt[ext] = st
		}
		st.files++
		st.totalPositions += totalPos
		st.spacePositions += spacePos
		st.tabPositions += tabPos
		st.spaceRunCount += spaceRuns
		st.tabRunCount += tabRuns
		st.spaceRunLenSum += spaceRunLen
		st.tabRunLenSum += tabRunLen
		st.totalBytes += int64(len(src))
		st.totalWhitespace += totalWS

		globalTotal += totalPos
		globalSpace += spacePos
		globalTab += tabPos
		globalWS += totalWS
		globalBytes += int64(len(src))
		globalSpaceRuns += spaceRuns
		globalTabRuns += tabRuns
		globalSpaceRunLen += spaceRunLen
		globalTabRunLen += tabRunLen

		return nil
	})

	exts := make([]string, 0, len(byExt))
	for ext := range byExt {
		exts = append(exts, ext)
	}
	sort.Slice(exts, func(i, j int) bool {
		return byExt[exts[i]].totalPositions > byExt[exts[j]].totalPositions
	})

	fmt.Println("Whitespace fraction of suffix array positions (sorted by total SA positions)")
	fmt.Println(strings.Repeat("=", 105))
	fmt.Printf("%-12s %5s %10s %10s %8s %8s %8s %8s %8s\n",
		"Extension", "Files", "SA Posns", "WS Posns", "WS%", "Space%", "Tab%", "SpRuns", "AvgRun")
	fmt.Println(strings.Repeat("-", 105))

	for _, ext := range exts {
		st := byExt[ext]
		if st.totalPositions == 0 {
			continue
		}
		wsPct := float64(st.spacePositions+st.tabPositions) / float64(st.totalPositions) * 100
		spacePct := float64(st.spacePositions) / float64(st.totalPositions) * 100
		tabPct := float64(st.tabPositions) / float64(st.totalPositions) * 100
		avgRun := float64(0)
		totalRuns := st.spaceRunCount + st.tabRunCount
		if totalRuns > 0 {
			avgRun = float64(st.spaceRunLenSum+st.tabRunLenSum) / float64(totalRuns)
		}
		fmt.Printf("%-12s %5d %10d %10d %7.1f%% %7.1f%% %7.1f%% %8d %8.1f\n",
			ext, st.files, st.totalPositions, st.spacePositions+st.tabPositions,
			wsPct, spacePct, tabPct, totalRuns, avgRun)
	}

	fmt.Println(strings.Repeat("-", 105))
	wsPct := float64(globalSpace+globalTab) / float64(globalTotal) * 100
	spacePct := float64(globalSpace) / float64(globalTotal) * 100
	tabPct := float64(globalTab) / float64(globalTotal) * 100
	totalRuns := globalSpaceRuns + globalTabRuns
	avgRun := float64(0)
	if totalRuns > 0 {
		avgRun = float64(globalSpaceRunLen+globalTabRunLen) / float64(totalRuns)
	}
	fmt.Printf("%-12s %5d %10d %10d %7.1f%% %7.1f%% %7.1f%% %8d %8.1f\n",
		"TOTAL", len(exts), globalTotal, globalSpace+globalTab,
		wsPct, spacePct, tabPct, totalRuns, avgRun)

	fmt.Println()
	fmt.Printf("Total file bytes: %d\n", globalBytes)
	fmt.Printf("Total SA positions: %d\n", globalTotal)
	fmt.Printf("Whitespace SA positions: %d (%.1f%%)\n", globalSpace+globalTab, wsPct)
	fmt.Printf("  Spaces: %d (%.1f%%)\n", globalSpace, spacePct)
	fmt.Printf("  Tabs: %d (%.1f%%)\n", globalTab, tabPct)
	fmt.Printf("Whitespace runs (len >= 2): %d\n", totalRuns)
	fmt.Printf("  Space runs: %d (avg len %.1f)\n", globalSpaceRuns, func() float64 {
		if globalSpaceRuns == 0 {
			return 0
		}
		return float64(globalSpaceRunLen) / float64(globalSpaceRuns)
	}())
	fmt.Printf("  Tab runs: %d (avg len %.1f)\n", globalTabRuns, func() float64 {
		if globalTabRuns == 0 {
			return 0
		}
		return float64(globalTabRunLen) / float64(globalTabRuns)
	}())
	fmt.Printf("Bytes in whitespace runs: %d\n", globalSpaceRunLen+globalTabRunLen)
	fmt.Printf("SA reduction if runs collapsed to 2 bytes each: %d → %d (%.1f%% smaller)\n",
		globalTotal, globalTotal-int64(globalSpaceRunLen+globalTabRunLen-2*(globalSpaceRuns+globalTabRuns)),
		float64(globalSpaceRunLen+globalTabRunLen-2*(globalSpaceRuns+globalTabRuns))/float64(globalTotal)*100)
}
