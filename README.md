# GomPDF

GomPDF is a small Go HTML/CSS to PDF engine with a focused MVP scope.
It is designed for simple documents, local assets, and a stable first release
that is useful from both Go code and the terminal.

## What It Supports

- HTML from strings or files
- Basic block layout and text flow
- Simple CSS cascade with tag, class, id, and inline styles
- Margins, padding, borders, colors, widths, heights, and text alignment
- Local images and SVG files
- Core PDF fonts: Helvetica, Times, and Courier
- A CLI that converts `input.html` to `output.pdf`

## What It Does Not Promise Yet

- JavaScript
- Flexbox or grid
- Full browser compatibility
- Custom font discovery or broad CSS3 coverage

## Install

```bash
go get github.com/gompdf/gompdf
go install github.com/gompdf/gompdf/cmd/gompdf@latest
```

## Library Usage

Convert an HTML string:

```go
package main

import (
	"log"

	"github.com/gompdf/gompdf"
)

func main() {
	html := `<!doctype html>
<html>
  <body>
    <h1>Hello, GomPDF</h1>
    <p>This document uses the MVP subset.</p>
  </body>
</html>`

	converter := gompdf.New()
	if err := converter.ConvertToFile(html, "output.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

Convert an HTML file:

```go
package main

import (
	"log"

	"github.com/gompdf/gompdf"
)

func main() {
	converter := gompdf.NewWithOptions(gompdf.DefaultOptions())
	if err := converter.ConvertFile("input.html", "output.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

## CLI Usage

```bash
gompdf input.html output.pdf
gompdf -v input.html output.pdf
gompdf -i input.html -o output.pdf
```

If no output path is provided, the CLI writes `<input>.pdf`.

## Examples

- [Minimal](examples/minimal/)
- [Images and styles](examples/images_and_styles/)

## Docs

- [Usage guide](docs/usage.md)
- [How it works](docs/how-it-works.md)
- [Roadmap](docs/roadmap.md)
