package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var version = "<dev>"

//goland:noinspection GoUnhandledErrorResult
func usage() {
	fmt.Fprintf(os.Stderr, "mdunwrap %s\n", version)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Remove unnecessary line breaks from Markdown, for readers whose viewer wraps text itself.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  mdunwrap [options] [file]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Either -in (or a bare filename) or -in-place is required. A path of '-' means stdin/stdout;")
	fmt.Fprintln(os.Stderr, "-out defaults to stdout. -in-place cannot be combined with -in or -out.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Only line breaks that soft-wrap prose are removed. Headings, list items, table rows, code")
	fmt.Fprintln(os.Stderr, "blocks, HTML blocks, frontmatter, and explicit hard breaks (a line ending in two spaces or")
	fmt.Fprintln(os.Stderr, "a backslash) are left exactly as they were.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "mdunwrap is written by Chris Dzombak <https://www.dzombak.com> and licensed under the Apache-2.0 license.")
	fmt.Fprintln(os.Stderr, "🌐 https://www.github.com/cdzombak/mdunwrap")
}

func main() {
	log.SetFlags(0)

	inPath := flag.String("in", "", "Path to the input file, or '-' for stdin.")
	outPath := flag.String("out", "", "Path to the output file, or '-' for stdout. Defaults to stdout.")
	inPlace := flag.String("in-place", "", "Path to a file to rewrite in place.")
	force := flag.Bool("force", false, "Allow -out to overwrite an existing file.")
	printVersion := flag.Bool("version", false, "Print version and exit.")
	flag.Usage = usage
	flag.Parse()

	if *printVersion {
		fmt.Printf("mdunwrap %s\n", version)
		os.Exit(0)
	}

	in, out := resolvePaths(*inPath, *outPath, *inPlace)

	if *inPlace != "" {
		if err := rewriteInPlace(*inPlace); err != nil {
			log.Fatalf("%s: %s", *inPlace, err)
		}
		return
	}
	if err := transform(in, out, *force); err != nil {
		log.Fatalf("%s", err)
	}
}

// resolvePaths validates the flag combination and returns the input and output
// paths to use, where "-" means stdin/stdout.
func resolvePaths(inPath, outPath, inPlace string) (in, out string) {
	args := flag.Args()

	if inPlace != "" {
		if inPath != "" || outPath != "" {
			log.Fatalf("-in-place cannot be combined with -in or -out")
		}
		if len(args) > 0 {
			log.Fatalf("-in-place cannot be combined with a filename argument")
		}
		return "", ""
	}

	switch {
	case len(args) > 1:
		log.Fatalf("expected at most one filename argument; got %d", len(args))
	case len(args) == 1 && inPath != "":
		log.Fatalf("cannot use both -in and a filename argument")
	case len(args) == 1:
		in = args[0]
	case inPath != "":
		in = inPath
	default:
		usage()
		os.Exit(1)
	}

	out = outPath
	if out == "" {
		out = "-"
	}
	return in, out
}

func transform(in, out string, force bool) error {
	src, err := readSource(in)
	if err != nil {
		return err
	}
	result, err := Unwrap(src)
	if err != nil {
		return err
	}
	return writeResult(out, result, force)
}

func readSource(path string) ([]byte, error) {
	if path == "-" {
		src, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		return src, nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return src, nil
}

func writeResult(path string, result []byte, force bool) error {
	if path == "-" {
		if _, err := os.Stdout.Write(result); err != nil {
			return fmt.Errorf("writing stdout: %w", err)
		}
		return nil
	}

	// O_EXCL makes the refusal to overwrite atomic, rather than a racy stat.
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists; pass -force to overwrite it", path)
		}
		return fmt.Errorf("opening %s: %w", path, err)
	}
	if _, err := f.Write(result); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	return nil
}

// rewriteInPlace replaces path with its unwrapped content, writing to a
// temporary file in the same directory and renaming over the original so a
// failure part-way through cannot truncate the user's document.
func rewriteInPlace(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory")
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	result, err := Unwrap(src)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(result); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}
	if err := os.Chmod(tmpName, info.Mode().Perm()); err != nil {
		return fmt.Errorf("setting mode on temporary file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replacing original: %w", err)
	}
	return nil
}
