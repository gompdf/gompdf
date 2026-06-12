# Usage

This guide describes the MVP subset that GomPDF supports today.

## CLI

Convert a file:

```bash
gompdf input.html output.pdf
```

If you omit the output path, GomPDF writes `<input>.pdf`:

```bash
gompdf input.html
```

The CLI also accepts `-i/-input`, `-o/-output`, and `-v/-verbose`.

## Library

Convert an HTML string directly to PDF:

```go
package main

import (
	"log"

	"github.com/gompdf/gompdf"
)

func main() {
	converter := gompdf.New()
	html := `<!doctype html>
<html>
  <body>
    <h1>Hello</h1>
    <p>This is a simple document.</p>
  </body>
</html>`

	if err := converter.ConvertToFile(html, "output.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

Convert a file:

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

## Supported Subset

- Elements: `html`, `head`, `body`, `div`, `p`, headings, lists, tables, `img`, `span`, `strong`, and `em`
- CSS selectors: tag, class, id, descendant, and inline styles
- CSS properties: margin, padding, border, color, background-color, font-family, font-size, font-style, font-weight, line-height, text-align, width, and height
- Resources: local files, data URLs, SVG, and basic raster images
- Fonts: core PDF fonts only

## Not Supported

- JavaScript
- Flexbox
- Grid
- Media queries
- Browser-level layout compatibility
- Custom font discovery and embedding

## Tips

- Use `ConvertFile` when you have relative CSS or images.
- Use `ConvertToFile` for in-memory HTML and set `Options.ResourcePaths` if it references local assets.
- Stick to simple, predictable markup. Block elements, images, and tables are the safest starting point.
