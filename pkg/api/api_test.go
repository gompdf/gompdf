package api

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
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
