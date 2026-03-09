// Command example walks a directory tree, compresses each text source file
// with boardgame, and reports per-extension and overall compression ratios.
//
// Usage:
//
//	go run ./example /path/to/project
//	go run ./example -exclude vendor /path/to/project
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"boardgame"
)

func main() {
	exclude := flag.String("exclude", "", "directory name to exclude (in addition to .git and the example dir itself)")
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./example [-exclude dir] <directory>")
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
	// When run via "go run ./example", the working dir is the module root,
	// so selfDir is the module root. We want to skip the example/ subtree.
	exampleDir := filepath.Join(selfDir, "example")

	type extStats struct {
		files      int
		origTotal  int
		compTotal  int
		ratioSum   float64
	}
	byExt := make(map[string]*extStats)
	var totalFiles int
	var totalOrig, totalComp int
	var totalRatioSum float64

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" {
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
		if err != nil {
			return err
		}
		if len(src) == 0 {
			return nil
		}

		compressed, err := boardgame.Encode(src)
		if err != nil {
			// skip files that contain bytes outside the valid range
			return nil
		}

		origLen, compLen, ratio := boardgame.Stats(src, compressed)

		st := byExt[ext]
		if st == nil {
			st = &extStats{}
			byExt[ext] = st
		}
		st.files++
		st.origTotal += origLen
		st.compTotal += compLen
		st.ratioSum += ratio

		totalFiles++
		totalOrig += origLen
		totalComp += compLen
		totalRatioSum += ratio

		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if totalFiles == 0 {
		fmt.Println("no compressible files found")
		return
	}

	// Sort extensions alphabetically.
	exts := make([]string, 0, len(byExt))
	for ext := range byExt {
		exts = append(exts, ext)
	}
	sort.Strings(exts)

	fmt.Printf("%-12s %6s %12s %12s %10s\n", "Extension", "Files", "Original", "Compressed", "Avg Ratio")
	fmt.Println(strings.Repeat("-", 56))
	for _, ext := range exts {
		st := byExt[ext]
		avg := st.ratioSum / float64(st.files)
		fmt.Printf("%-12s %6d %12d %12d %9.1f%%\n", ext, st.files, st.origTotal, st.compTotal, avg*100)
	}
	fmt.Println(strings.Repeat("-", 56))
	avg := totalRatioSum / float64(totalFiles)
	fmt.Printf("%-12s %6d %12d %12d %9.1f%%\n", "TOTAL", totalFiles, totalOrig, totalComp, avg*100)
}
