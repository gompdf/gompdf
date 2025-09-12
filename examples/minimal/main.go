package main

import (
	"fmt"
	"log"

	"github.com/gompdf/gompdf"
)

func main() {
	html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>Hello GomPDF</title>
  <style>
    :root { --ink: #1f2937; --muted: #6b7280; }
    body { font-family: Helvetica, Arial, sans-serif; color: var(--ink); margin: 0; }
    .page { padding: 32px; }
    h1 { margin: 0 0 12px; font-size: 28px; }
    p { margin: 0 0 10px; line-height: 1.5; }
    .box { border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; }
    .muted { color: var(--muted); }
  </style>
</head>
<body>
  <div class="page">
    <h1>Hello, GomPDF!</h1>
    <p class="muted">This is a minimal example converting inline HTML to a PDF.</p>
    <div class="box">
      <p>• Page size: A4 (default)</p>
      <p>• Margins: 36pt</p>
      <p>• Rendering: Basic text, borders and spacing</p>
    </div>
  </div>
</body>
</html>`

	// Configure some common defaults
	opts := gompdf.DefaultOptions()
	opts.MarginTop = 36
	opts.MarginRight = 36
	opts.MarginBottom = 36
	opts.MarginLeft = 36
	converter := gompdf.NewWithOptions(opts)

	output := "hello.pdf"
	if err := converter.ConvertToFile(html, output); err != nil {
		log.Fatalf("convert failed: %v", err)
	}

	fmt.Printf("Wrote %s\n", output)
}
