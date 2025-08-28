# How GomPDF Works

This doc summarizes the core pipeline from HTML/CSS input to a PDF output and shows how the examples use it.

## Overview
- __Inputs__: HTML (file or string). CSS can be inline (`<style>`) or linked via `<link>`.
- __Pipeline__: Parse HTML → Parse CSS → Apply cascade → Layout → Pagination → Render to PDF.
- __APIs__: `pkg/api/` exposes `ConvertFile`, `ConvertToFile`, and `Convert` via a `Converter`.
- __Resources__: Relative paths (e.g., linked CSS) resolve using `Options.ResourcePaths` and/or the source file directory.

## Stages
- __HTML parsing__: `internal/parser/html/` produces a DOM-like tree (`html.Node`).
- __CSS parsing & cascade__: `internal/parser/css/` parses rules; `internal/style/cascade.go` matches selectors and applies declarations.
- __Layout__: `internal/layout/` builds block/inline boxes and computes geometry.
- __Pagination__: `internal/pagination/` splits content across pages.
- __PDF rendering__: `internal/render/pdf/` writes pages, text, borders, and backgrounds to the PDF.

## Options (subset)
Defined in `pkg/api/options.go` and used by `api.NewWithOptions(...)`:
- __Page size/margins__: `PageWidth`, `PageHeight`, `MarginTop/Right/Bottom/Left`.
- __Page orientation__: `PageOrientation` (portrait or landscape) controls effective page dimensions.
- __Rendering flags__: `RenderBackgrounds`, `RenderBorders`.
- __Debug__: `Debug`, `DebugDrawBoxes`.
- __Resources__: `ResourcePaths` for resolving relative URLs (e.g., CSS files, images).

## Using templates
GomPDF does not execute PHP; use Go templates (or any preprocessor) to render HTML before conversion.
- File-based flow: render template → write `.html` → `ConvertFile(htmlPath, pdfPath)`.
- In-memory flow: render template → `ConvertToFile(htmlString, pdfPath)` and ensure `ResourcePaths` cover any linked assets.

## Examples
- __Styled demo__ (`examples/styled-demo/`)
  - Renders `styled.tmpl.html` with `html/template` to `styled.html`.
  - Uses external `styled.css`. `opts.ResourcePaths` includes the working directory so the link resolves.
  - Converts via `conv.ConvertFile("styled.html", "styled.pdf")` with backgrounds/borders enabled to show color swatches and boxes.

- __Invoice basic__ (`examples/invoice-basic/`)
  - Renders `invoice.tmpl.html` with computed totals to `invoice.html`.
  - Uses inline `<style>`.
  - Converts via `converter.ConvertFile(...)`. Backgrounds/borders enabled for header shading and lines.

## File vs. string inputs
- __File path__: Simplifies relative path handling (base directory is known).
- __String input__: Use `ConvertToFile(html, pdfPath)` or `Convert(html)` but set `Options.ResourcePaths` to resolve any relative links.

## CLI
- A simple CLI lives in `cmd/gompdf/`. It wraps the same converter.

## Page Orientation
GomPDF supports both portrait and landscape page orientations:

- __Configuration__: Set via `Options.PageOrientation` (values: `PageOrientationPortrait` or `PageOrientationLandscape`).
- __Functional option__: Use `WithPageOrientation(PageOrientationLandscape)` to set orientation.
- __Processing__: 
  1. The converter determines effective page dimensions based on orientation in `api.ConvertToFile()`.
  2. If landscape is specified and width < height, dimensions are swapped.
  3. If portrait is specified and width > height, dimensions are swapped.
  4. These effective dimensions are passed to layout and pagination engines.
- __PDF output__: The PDF renderer sets the orientation flag ("P" or "L") based on page dimensions.

This approach ensures that content is properly laid out and paginated according to the specified orientation, and that the PDF viewer displays the document in the correct orientation.

For more details, see `docs/design-overview.md` and `docs/getting-started.md`.
