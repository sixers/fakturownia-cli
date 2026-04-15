package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sixers/fakturownia-cli/internal/spec"
)

func main() {
	check := flag.Bool("check", false, "validate generated skill files without writing them")
	flag.Parse()

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *check {
		if err := spec.CheckSkillFiles(repoRoot); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := spec.GenerateSkillFiles(repoRoot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		goMod := filepath.Join(dir, "go.mod")
		raw, readErr := os.ReadFile(goMod)
		if readErr == nil && strings.Contains(string(raw), "module github.com/sixers/fakturownia-cli") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("could not locate fakturownia-cli repo root from %s", cwd)
}
