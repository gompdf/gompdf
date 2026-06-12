package api

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestConvertToFileProducesPDF(t *testing.T) {
	converter := New()
	outputPath := filepath.Join(t.TempDir(), "simple.pdf")

	html := `<!doctype html>
<html>
  <body>
    <h1>Hello, GomPDF</h1>
    <p>This is a simple PDF.</p>
  </body>
</html>`

	if err := converter.ConvertToFile(html, outputPath); err != nil {
		t.Fatalf("ConvertToFile() error = %v", err)
	}

	assertPDFFile(t, outputPath)
}

func TestConvertFileWithLocalImage(t *testing.T) {
	dir := t.TempDir()

	imagePath := filepath.Join(dir, "pixel.png")
	if err := writeTestPNG(imagePath); err != nil {
		t.Fatalf("writeTestPNG() error = %v", err)
	}

	inputPath := filepath.Join(dir, "input.html")
	outputPath := filepath.Join(dir, "output.pdf")
	html := `<!doctype html>
<html>
  <body>
    <p style="font-family: Courier; font-size: 14px;">Core fonts are available.</p>
    <img src="pixel.png" style="width: 24px; height: 24px;" />
  </body>
</html>`
	if err := os.WriteFile(inputPath, []byte(html), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	converter := New()
	if err := converter.ConvertFile(inputPath, outputPath); err != nil {
		t.Fatalf("ConvertFile() error = %v", err)
	}

	assertPDFFile(t, outputPath)
}

func TestConvertToFileRendersStyledShorthands(t *testing.T) {
	converter := NewWithOptions(DefaultOptions())
	converter.options.RenderBackgrounds = true
	converter.options.RenderBorders = true
	outputPath := filepath.Join(t.TempDir(), "styled.pdf")

	html := `<!doctype html>
<html>
  <head>
    <style>
      body { margin: 0; }
      .header { border-bottom: 1px solid #e5e7eb; padding: 24px 0; }
      .card { border: 1px solid #e5e7eb; background: #f9fafb; padding: 16px; }
      .footer { border-top: 1px solid #e5e7eb; color: #6b7280; }
    </style>
  </head>
  <body>
    <header class="header">Header</header>
    <section class="card">Card</section>
    <footer class="footer">Footer</footer>
  </body>
</html>`

	if err := converter.ConvertToFile(html, outputPath); err != nil {
		t.Fatalf("ConvertToFile() error = %v", err)
	}

	assertPDFFile(t, outputPath)
	streams := extractPDFStreams(t, outputPath)
	if !strings.Contains(streams, pdfColorCommand(249, 250, 251, "rg")) {
		t.Fatalf("expected card background fill color in PDF streams:\n%s", streams)
	}
	if !strings.Contains(streams, pdfColorCommand(229, 231, 235, "RG")) {
		t.Fatalf("expected border stroke color in PDF streams:\n%s", streams)
	}
}

func TestConvertToFileRespectsConfiguredMargins(t *testing.T) {
	converter := NewWithOptions(DefaultOptions())
	converter.options.MarginTop = 10
	converter.options.MarginRight = 20
	converter.options.MarginBottom = 30
	converter.options.MarginLeft = 40
	outputPath := filepath.Join(t.TempDir(), "margins.pdf")

	html := `<!doctype html>
<html>
  <body>
    <div style="font-size: 11px; margin: 0;">Hello margins</div>
  </body>
</html>`

	if err := converter.ConvertToFile(html, outputPath); err != nil {
		t.Fatalf("ConvertToFile() error = %v", err)
	}

	assertPDFFile(t, outputPath)
	streams := extractPDFStreams(t, outputPath)
	x, ok := firstPDFTextX(streams, "Hello margins")
	if !ok {
		t.Fatalf("did not find the expected text in PDF streams:\n%s", streams)
	}
	if x < 38 || x > 42 {
		t.Fatalf("expected text to start near the configured left margin, got x=%.2f", x)
	}
}

func assertPDFFile(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected a non-empty PDF file")
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("file does not look like a PDF: %q", data[:min(len(data), 8)])
	}
	if !bytes.Contains(data, []byte("%%EOF")) {
		t.Fatal("PDF footer marker not found")
	}
}

func writeTestPNG(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 0x22, G: 0x55, B: 0xaa, A: 0xff})
	img.Set(1, 0, color.RGBA{R: 0x22, G: 0x55, B: 0xaa, A: 0xff})
	img.Set(0, 1, color.RGBA{R: 0x22, G: 0x55, B: 0xaa, A: 0xff})
	img.Set(1, 1, color.RGBA{R: 0x22, G: 0x55, B: 0xaa, A: 0xff})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func extractPDFStreams(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var out strings.Builder
	for {
		start := bytes.Index(data, []byte("stream"))
		if start == -1 {
			break
		}
		data = data[start+len("stream"):]
		if len(data) > 0 && data[0] == '\r' {
			data = data[1:]
		}
		if len(data) > 0 && data[0] == '\n' {
			data = data[1:]
		}

		end := bytes.Index(data, []byte("endstream"))
		if end == -1 {
			break
		}

		chunk := bytes.TrimRight(data[:end], "\r\n")
		if zr, err := zlib.NewReader(bytes.NewReader(chunk)); err == nil {
			if decoded, readErr := io.ReadAll(zr); readErr == nil {
				out.Write(decoded)
			}
			zr.Close()
		}

		data = data[end+len("endstream"):]
	}

	return out.String()
}

func pdfColorCommand(r, g, b int, op string) string {
	return strings.Join([]string{
		formatPDFColor(r),
		formatPDFColor(g),
		formatPDFColor(b),
		op,
	}, " ")
}

func formatPDFColor(v int) string {
	return fmt.Sprintf("%.3f", float64(v)/255)
}

func firstPDFTextX(streams, text string) (float64, bool) {
	pattern := regexp.MustCompile(`(?s)BT\s+([0-9.]+)\s+([0-9.]+)\s+Td\s+\(` + regexp.QuoteMeta(text) + `\)\s+Tj`)
	match := pattern.FindStringSubmatch(streams)
	if match == nil {
		return 0, false
	}

	x, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, false
	}

	return x, true
}
