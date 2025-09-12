package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gompdf/gompdf"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}

	opts := gompdf.DefaultOptions()
	opts.MarginTop = 36
	opts.MarginRight = 36
	opts.MarginBottom = 36
	opts.MarginLeft = 36
	// Allow relative resources like styles.css and logo.svg to be discovered
	opts.ResourcePaths = []string{cwd}

	conv := gompdf.NewWithOptions(opts)

	input := filepath.Join(cwd, "logo_svg.html")
	output := filepath.Join(cwd, "logo_svg.pdf")
	if err := conv.ConvertFile(input, output); err != nil {
		log.Fatalf("convert file failed: %v", err)
	}
	fmt.Printf("Wrote %s\n", output)
}
