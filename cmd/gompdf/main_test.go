package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunConvertsPositionalArgs(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.html")
	outputPath := filepath.Join(dir, "output.pdf")

	html := `<!doctype html>
<html>
  <body>
    <h1>CLI test</h1>
    <p>GomPDF should render this file.</p>
  </body>
</html>`
	if err := os.WriteFile(inputPath, []byte(html), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := run([]string{inputPath, outputPath}, io.Discard, io.Discard); err != nil {
		t.Fatalf("run() error = %v", err)
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
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
