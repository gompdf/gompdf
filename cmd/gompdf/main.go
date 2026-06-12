package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gompdf/gompdf"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("gompdf", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: gompdf [options] <input.html> [output.pdf]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Options:")
		fs.PrintDefaults()
	}

	var inputFile string
	var outputFile string
	var verbose bool

	fs.StringVar(&inputFile, "input", "", "Input HTML file path")
	fs.StringVar(&inputFile, "i", "", "Input HTML file path")
	fs.StringVar(&outputFile, "output", "", "Output PDF file path")
	fs.StringVar(&outputFile, "o", "", "Output PDF file path")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&verbose, "v", false, "Enable verbose logging")

	if err := fs.Parse(args); err != nil {
		return err
	}

	rest := fs.Args()
	if inputFile == "" && len(rest) > 0 {
		inputFile = rest[0]
	}
	if outputFile == "" && len(rest) > 1 {
		outputFile = rest[1]
	}
	if len(rest) > 2 {
		return fmt.Errorf("unexpected extra arguments: %v", rest[2:])
	}

	if inputFile == "" {
		fs.Usage()
		return errors.New("input file is required")
	}

	if outputFile == "" {
		outputFile = deriveOutputPath(inputFile)
	}

	converter := gompdf.New()
	if verbose {
		converter = converter.SetDebug(true)
	}

	if err := converter.ConvertFile(inputFile, outputFile); err != nil {
		return err
	}

	if verbose {
		fmt.Fprintf(stdout, "Successfully converted %s to %s\n", inputFile, outputFile)
	}

	return nil
}

func deriveOutputPath(inputFile string) string {
	ext := filepath.Ext(inputFile)
	if ext == "" {
		return inputFile + ".pdf"
	}
	return inputFile[:len(inputFile)-len(ext)] + ".pdf"
}
