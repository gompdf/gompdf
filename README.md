# GomPDF

<img width="500" height="500" alt="1" src="https://github.com/user-attachments/assets/eab81250-47cd-4931-a7bd-048e90e599b9" />

---

A native HTML/CSS → PDF engine in Go, focused on CSS 2.1 subset, robust typography (Harfbuzz), and server safety.

[![Go Report Card](https://goreportcard.com/badge/github.com/henrrius/gompdf)](https://goreportcard.com/report/github.com/henrrius/gompdf)
[![GoDoc](https://godoc.org/github.com/henrrius/gompdf?status.svg)](https://godoc.org/github.com/henrrius/gompdf)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Status
🚧 Experimental. v0.1 targets a minimal printable subset (see [roadmap](docs/roadmap.md)).

## Features

- HTML parsing with support for most common elements
- CSS styling with cascade, inheritance, and specificity
- Text layout with proper line breaking and justification
- Bidirectional text support (RTL languages)
- Page pagination with headers and footers
- PDF generation with embedded fonts and images
- Command-line tool for easy conversion

## Install

### Library

```bash
go get github.com/henrrius/gompdf
```

### CLI Tool

```bash
go install github.com/henrrius/gompdf/cmd/gompdf@latest
```

## Usage

### As a Library

```go
package main

import (
	"log"

	"github.com/henrrius/gompdf/pkg/api"
)

func main() {
	// Create a new converter with default options
	converter := api.New()

	// Convert HTML file to PDF
	err := converter.ConvertFile("input.html", "output.pdf")
	if err != nil {
		log.Fatalf("Error converting HTML to PDF: %v", err)
	}
}
```

### With Custom Options

```go
package main

import (
	"log"

	"github.com/henrrius/gompdf/pkg/api"
)

func main() {
	// Create a converter with custom options
	converter := api.NewWithOptions(
		api.Options{
			PageWidth:  api.PageSizeA4Width,
			PageHeight: api.PageSizeA4Height,
			// Set page orientation (portrait is default)
			PageOrientation: api.PageOrientationPortrait,
			MarginTop:  72,  // 1 inch in points
			MarginLeft: 72,
			MarginRight: 72,
			MarginBottom: 72,
			Title:    "My Document",
			Author:   "GomPDF",
		}
	)

	// Add font directories
	converter.AddFontDirectory("/path/to/fonts")

	// Convert HTML string to PDF
	html := `<!DOCTYPE html>
<html>
<head>
  <title>Hello World</title>
</head>
<body>
  <h1>Hello, GomPDF!</h1>
  <p>This is a simple HTML document.</p>
</body>
</html>`

	err := converter.ConvertToFile(html, "output.pdf")
	if err != nil {
		log.Fatalf("Error converting HTML to PDF: %v", err)
	}
}
```

### Using Functional Options

```go
package main

import (
	"log"

	"github.com/henrrius/gompdf"
)

func main() {
	// Create a new converter with default options
	converter := gompdf.New()

	// Apply functional options
	converter = converter.WithOption(
		// Set page size to Letter
		gompdf.WithPageSize(gompdf.PageSizeLetterWidth, gompdf.PageSizeLetterHeight),
	).WithOption(
		// Set landscape orientation
		gompdf.WithPageOrientation(gompdf.PageOrientationLandscape),
	).WithOption(
		// Set custom margins
		gompdf.WithMargins(36, 36, 36, 36), // 0.5 inch margins
	)

	// Convert HTML to PDF
	err := converter.ConvertFile("input.html", "landscape-letter.pdf")
	if err != nil {
		log.Fatalf("Error converting HTML to PDF: %v", err)
	}
}
```

### Using the CLI

```bash
# Convert an HTML file to PDF
gompdf -i input.html -o output.pdf

# Enable verbose logging
gompdf -i input.html -o output.pdf -v
```

## Documentation

- [Getting Started](docs/getting-started.md)
- [Design Overview](docs/design-overview.md)
- [Roadmap](docs/roadmap.md)

## Examples

Check the [examples](examples/) directory for more usage examples.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
