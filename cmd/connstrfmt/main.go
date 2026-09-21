// Command connstrfmt validates and reformats a connection string, either
// from a file given as an argument or from stdin.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	connstr "github.com/nancy-m1989/connstr-fmt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	validate := flag.Bool("validate", false, "report every error in the input instead of stopping at the first one, then exit")
	write := flag.Bool("w", false, "write the formatted result back to the input file instead of printing to stdout")
	flag.Parse()

	path := flag.Arg(0)
	input, err := readInput(path)
	if err != nil {
		return err
	}

	if *validate {
		if *write {
			return fmt.Errorf("-w cannot be combined with -validate")
		}
		return runValidate(input)
	}

	cs, err := connstr.ParseAny(input)
	if err != nil {
		return err
	}
	formatted := cs.Format()

	if *write {
		return writeFile(path, formatted)
	}

	fmt.Println(formatted)
	return nil
}

// writeFile rewrites path with formatted, preserving the file's existing
// permission bits. path must be non-empty: -w on stdin input has nowhere to
// write back to.
func writeFile(path, formatted string) error {
	if path == "" {
		return fmt.Errorf("-w requires a file argument, reading from stdin has nowhere to write back to")
	}

	perm := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	if err := os.WriteFile(path, []byte(formatted+"\n"), perm); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// runValidate prints every error found in input and reports failure via a
// non-nil error if there was at least one, so main sets a non-zero exit
// status without duplicating the last error onto stderr.
func runValidate(input string) error {
	errs := connstr.Validate(input)
	if len(errs) == 0 {
		fmt.Println("ok")
		return nil
	}
	for _, e := range errs[:len(errs)-1] {
		fmt.Fprintln(os.Stderr, e)
	}
	return errs[len(errs)-1]
}

func readInput(path string) (string, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", path, err)
		}
		return string(data), nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	return string(data), nil
}
