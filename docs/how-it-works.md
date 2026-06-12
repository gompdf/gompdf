# How It Works

GomPDF uses a straight pipeline:

1. Parse HTML into a document tree.
2. Parse CSS and resolve the cascade.
3. Build a simple layout tree for blocks, text, images, and tables.
4. Paginate the laid out boxes into PDF pages.
5. Render the pages through `fpdf`.

## Supported Flow

- `ConvertFile` reads HTML from disk and resolves relative assets from the source directory.
- `ConvertToFile` converts an in-memory HTML string and writes a PDF file.
- `Convert` writes PDF bytes to an `io.Writer`.

## Scope

The MVP is intentionally small:

- No JavaScript
- No flexbox or grid
- No browser-grade CSS compatibility
- Core fonts only

For examples and copy-paste usage, see [Usage](usage.md).
