package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const minCoverage = 90.0

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: check_coverage <coverprofile>")
		os.Exit(2)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()

	totalStmts := 0
	coveredStmts := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		fileName := strings.ReplaceAll(parts[0], `\`, `/`)
		if shouldSkip(fileName) {
			continue
		}
		stmts, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		totalStmts += stmts
		if count > 0 {
			coveredStmts += stmts
		}
	}
	if totalStmts == 0 {
		fmt.Println("no statements to measure")
		os.Exit(0)
	}
	pct := float64(coveredStmts) / float64(totalStmts) * 100
	fmt.Printf("business logic coverage: %.1f%% (%d/%d)\n", pct, coveredStmts, totalStmts)
	if pct+0.05 < minCoverage {
		fmt.Fprintf(os.Stderr, "coverage below %.0f%%\n", minCoverage)
		os.Exit(1)
	}
}

func shouldSkip(path string) bool {
	skip := []string{
		"/cmd/",
		"/pkg/db/",
		"/internal/oidc/",
		"main.go",
	}
	for _, part := range skip {
		if strings.Contains(path, part) {
			return true
		}
	}
	return false
}
