package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gompdf/gompdf"
)

func main() {
	var (
		inputURL  = flag.String("url", "https://example.com", "URL to fetch and convert to PDF")
		outputPDF = flag.String("o", "page.pdf", "output PDF filename")
	)
	flag.Parse()

	opts := gompdf.DefaultOptions()
	// Example: set Letter portrait with 0.5in margins
	opts.PageWidth = gompdf.PageSizeA4Width
	opts.PageHeight = gompdf.PageSizeA4Height
	opts.PageOrientation = gompdf.PageOrientationLandscape
	opts.MarginTop = 18
	opts.MarginRight = 18
	opts.MarginBottom = 18
	opts.MarginLeft = 18

	conv := gompdf.NewWithOptions(opts)
	if err := conv.ConvertURL(*inputURL, *outputPDF); err != nil {
		log.Fatalf("convert URL failed: %v", err)
	}
	fmt.Printf("Converted %s -> %s\n", *inputURL, *outputPDF)
}
