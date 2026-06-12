# Examples

These examples stay inside the MVP subset that GomPDF supports today.

## Minimal

Converts an inline HTML string to PDF.

```bash
cd examples/minimal
go run .
```

## Images and Styles

Loads local CSS and a local image or SVG from disk.

```bash
cd examples/images_and_styles
go run .
```

## Experimental

The repository still contains older or more ambitious examples under
`examples/invoice/`, `examples/url_to_pdf/`, and `examples/user_report/`.
They are kept for reference, but they are not part of the small MVP promise.
