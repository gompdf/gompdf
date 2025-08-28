package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gompdf/gompdf"
)

func main() {
	var (
		inputFile  string
		outputFile string
		verbose    bool
	)

	flag.StringVar(&inputFile, "input", "", "Input HTML file path")
	flag.StringVar(&outputFile, "output", "", "Output PDF file path")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if inputFile == "" {
		fmt.Println("Error: input file is required")
		flag.Usage()
		os.Exit(1)
	}

	if outputFile == "" {
		ext := filepath.Ext(inputFile)
		outputFile = inputFile[:len(inputFile)-len(ext)] + ".pdf"
	}

	converter := gompdf.New()

	if verbose {
		converter = converter.SetDebug(true)
	}
	err := converter.ConvertFile(inputFile, outputFile)
	if err != nil {
		fmt.Printf("Error converting file: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Successfully converted %s to %s\n", inputFile, outputFile)
	}
}
