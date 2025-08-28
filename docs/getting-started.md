# Getting Started with GomPDF

GomPDF is a Go library for converting HTML/CSS documents to PDF. This guide will help you get started with using GomPDF in your projects.

## Installation

To install GomPDF, use the `go get` command:

```bash
go get github.com/henrrius/gompdf
```

## Basic Usage

### As a Library

```go
package main

import (
	"log"

	"github.com/henrrius/gompdf/pkg/api"
)

func main() {
	// Create a new converter with default options
	converter := api.New(api.Options{})

	// Convert an HTML file to PDF
	err := converter.ConvertFile("input.html", "output.pdf")
	if err != nil {
		log.Fatalf("Error converting file: %v", err)
	}

	// Or convert HTML string to PDF
	htmlContent := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Sample Document</title>
			<style>
				body { font-family: Arial, sans-serif; }
				h1 { color: #0066cc; }
			</style>
		</head>
		<body>
			<h1>Hello, GomPDF!</h1>
			<p>This is a sample document.</p>
		</body>
		</html>
	`
	
	err = converter.ConvertString(htmlContent, "string-output.pdf")
	if err != nil {
		log.Fatalf("Error converting string: %v", err)
	}
}
```

### As a Command Line Tool

GomPDF also comes with a command line tool that you can use to convert HTML files to PDF:

```bash
# Install the command line tool
go install github.com/henrrius/gompdf/cmd/gompdf@latest

# Convert an HTML file to PDF
gompdf --input input.html --output output.pdf

# Enable verbose logging
gompdf --input input.html --verbose
```

## Configuration Options

GomPDF provides various configuration options to customize the PDF output:

```go
options := api.Options{
    // Page dimensions
    PageWidth:  api.PageSizeA4Width,  // Set page width
    PageHeight: api.PageSizeA4Height, // Set page height
    
    // Page orientation (portrait or landscape)
    PageOrientation: api.PageOrientationPortrait, // Default is portrait
    
    // Page margins (in points, 72 points = 1 inch)
    MarginTop:    72,
    MarginRight:  72,
    MarginBottom: 72,
    MarginLeft:   72,
    
    // Rendering options
    DPI:   96,    // Set DPI (dots per inch)
    Debug: false, // Enable debug mode
    
    // Document metadata
    Title:    "My Document",
    Author:   "GomPDF User",
    Subject:  "Sample Document",
    Keywords: "gompdf,pdf,html",
}

converter := api.New(options)
```

## Next Steps

- Check out the [examples](../examples/) directory for more usage examples
- Read the [design overview](design-overview.md) to understand how GomPDF works
- See the [roadmap](roadmap.md) for upcoming features and improvements
