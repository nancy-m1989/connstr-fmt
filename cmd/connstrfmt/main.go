// Command connstrfmt validates and reformats a connection string, either
// from a file given as an argument or from stdin.
package main

import (
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
	input, err := readInput()
	if err != nil {
		return err
	}

	cs, err := connstr.Parse(input)
	if err != nil {
		return err
	}

	fmt.Println(cs.Format())
	return nil
}

func readInput() (string, error) {
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", os.Args[1], err)
		}
		return string(data), nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	return string(data), nil
}
