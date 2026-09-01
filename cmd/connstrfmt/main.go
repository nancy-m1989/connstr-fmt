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
	flag.Parse()

	input, err := readInput(flag.Arg(0))
	if err != nil {
		return err
	}

	if *validate {
		return runValidate(input)
	}

	cs, err := connstr.Parse(input)
	if err != nil {
		return err
	}

	fmt.Println(cs.Format())
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
