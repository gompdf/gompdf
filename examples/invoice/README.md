# Simple Invoice

A lightweight invoice example using a Go HTML template and `ConvertFile`.

This example stays inside the current MVP subset: tables, text, borders, and a
small amount of background painting. It avoids flexbox and other unsupported
layout features so the PDF output stays close to the browser version.

## Run

```bash
go run main.go
```

It produces `invoice.html` and `invoice.pdf`.
