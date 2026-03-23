package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"github.com/phasecurve/readme-merge/internal/engine"
	"github.com/phasecurve/readme-merge/internal/hook"
	"github.com/phasecurve/readme-merge/internal/parser"
	"github.com/phasecurve/readme-merge/internal/remote"
	"github.com/phasecurve/readme-merge/internal/source"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type commandConfig struct {
	readmePath string
	reader     engine.FileReader
}

func resolveConfig(sourceRef, readme string) *commandConfig {
	dir, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	if err := source.ValidateSource(sourceRef, dir); err != nil {
		fatal(err)
	}

	readmePath := readme
	if readmePath == "" {
		readmePath, err = findReadme(dir)
		if err != nil {
			fatal(err)
		}
	}

	localResolver := source.NewResolver(sourceRef, dir)
	cacheDir := filepath.Join(dir, ".readme-merge", "cache")
	reader := remote.NewCompositeReader(localResolver, cacheDir)

	return &commandConfig{
		readmePath: readmePath,
		reader:     reader,
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	arg := os.Args[1]
	switch {
	case arg == "update":
		runUpdate(os.Args[2:])
	case arg == "check":
		runCheck(os.Args[2:])
	case arg == "hook":
		runHook(os.Args[2:])
	case arg == "version" || arg == "--version" || arg == "-v":
		fmt.Printf("readme-merge %s (%s, built %s)\n", version, commit, date)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: readme-merge <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  update    populate/refresh all code placeholders")
	fmt.Fprintln(os.Stderr, "  check     verify all placeholders are fresh (exit 1 if stale)")
	fmt.Fprintln(os.Stderr, "  hook      install/uninstall git pre-commit hook")
	fmt.Fprintln(os.Stderr, "  version   print version information")
}

func findReadme(dir string) (string, error) {
	candidates := []string{"README.md", "readme.md", "Readme.md"}
	for _, c := range candidates {
		path := filepath.Join(dir, c)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no README.md found in %s", dir)
}

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	sourceRef := fs.String("source", "", "source ref: staged, HEAD, or git ref (default: worktree)")
	readme := fs.String("file", "", "path to README (default: auto-detect)")
	fs.Parse(args)

	cfg := resolveConfig(*sourceRef, *readme)

	content, err := os.ReadFile(cfg.readmePath)
	if err != nil {
		fatal(fmt.Errorf("reading README: %w", err))
	}

	result, err := engine.Update(string(content), cfg.reader)
	if err != nil {
		fatal(err)
	}

	if err := os.WriteFile(cfg.readmePath, []byte(result.Output), 0644); err != nil {
		fatal(fmt.Errorf("writing README: %w", err))
	}

	for _, u := range result.Unreachable {
		fmt.Fprintf(os.Stderr, "warning: %s (skipped, existing content preserved)\n", u.Message)
	}

	fmt.Printf("updated %d placeholder(s)\n", result.Updated)
}

func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	sourceRef := fs.String("source", "", "source ref: staged, HEAD, or git ref (default: worktree)")
	readme := fs.String("file", "", "path to README (default: auto-detect)")
	heal := fs.Bool("heal", false, "write healed line references back to README")
	full := fs.Bool("full", false, "show full content of each placeholder")
	fs.Parse(args)

	cfg := resolveConfig(*sourceRef, *readme)

	content, err := os.ReadFile(cfg.readmePath)
	if err != nil {
		fatal(fmt.Errorf("reading README: %w", err))
	}

	result, err := engine.Check(string(content), cfg.reader)
	if err != nil {
		fatal(err)
	}

	exitCode := 0

	if len(result.Unreachable) > 0 {
		fmt.Fprintf(os.Stderr, "%d placeholder(s) reference unreachable refs (branch deleted or renamed?):\n", len(result.Unreachable))
		for _, u := range result.Unreachable {
			fmt.Fprintf(os.Stderr, "  %s ref=%s lines %d-%d\n", u.Block.From, u.Block.Ref, u.Block.SourceStart, u.Block.SourceEnd)
			fmt.Fprintf(os.Stderr, "    %s\n", u.Message)
		}
		exitCode = 1
	}

	if len(result.Unhashed) > 0 {
		fmt.Fprintf(os.Stderr, "%d unhashed placeholder(s) - run 'readme-merge update' first:\n", len(result.Unhashed))
		for _, b := range result.Unhashed {
			printBlock(b, *full)
		}
		exitCode = 1
	}

	if len(result.Stale) > 0 {
		fmt.Fprintf(os.Stderr, "%d stale placeholder(s):\n", len(result.Stale))
		for _, s := range result.Stale {
			printBlock(s.Block, *full)
		}
		exitCode = 1
	}

	if result.Healed > 0 {
		if *heal {
			fmt.Printf("self-healed %d placeholder(s) (lines shifted)\n", result.Healed)
		} else {
			fmt.Printf("%d placeholder(s) have shifted lines (run with --heal to update)\n", result.Healed)
		}
	}

	if *heal && result.Output != "" {
		if err := os.WriteFile(cfg.readmePath, []byte(result.Output), 0644); err != nil {
			fatal(fmt.Errorf("writing README: %w", err))
		}
	}

	if len(result.FreshBlocks) > 0 {
		fmt.Printf("%d placeholder(s) fresh:\n", result.Fresh)
		for _, b := range result.FreshBlocks {
			printBlock(b, *full)
		}
	} else {
		fmt.Printf("%d placeholder(s) fresh\n", result.Fresh)
	}

	os.Exit(exitCode)
}

func runHook(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: readme-merge hook <install|uninstall>")
		os.Exit(1)
	}

	dir, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	switch args[0] {
	case "install":
		if err := hook.Install(dir); err != nil {
			fatal(err)
		}
		fmt.Println("pre-commit hook installed")
	case "uninstall":
		if err := hook.Uninstall(dir); err != nil {
			fatal(err)
		}
		fmt.Println("pre-commit hook removed")
	default:
		fmt.Fprintln(os.Stderr, "usage: readme-merge hook <install|uninstall>")
		os.Exit(1)
	}
}

func printBlock(b parser.Block, full bool) {
	label := b.From
	if b.Ref != "" {
		label += " ref=" + b.Ref
	}
	label += fmt.Sprintf(" lines %d-%d", b.SourceStart, b.SourceEnd)
	fmt.Printf("  %s\n", label)

	content := strings.TrimRight(b.Content, "\n")
	lines := strings.Split(content, "\n")

	if full {
		for _, l := range lines {
			fmt.Printf("    %s\n", l)
		}
	} else {
		limit := 3
		if len(lines) < limit {
			limit = len(lines)
		}
		for _, l := range lines[:limit] {
			fmt.Printf("    %s\n", l)
		}
		if len(lines) > 3 {
			fmt.Printf("    ... (%d more lines)\n", len(lines)-3)
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
