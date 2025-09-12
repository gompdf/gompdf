# Examples

A collection of small, focused examples showing how to use the `github.com/gompdf/gompdf` API.

- Minimal
  - Directory: `examples/minimal/`
  - Shows converting inline HTML to a PDF using `ConvertToFile` with default A4 and simple margins.
  - Run:
    ```bash
    cd examples/minimal
    go run main.go
    ```

- URL to PDF
  - Directory: `examples/url_to_pdf/`
  - Fetches a web page via `ConvertURL` and saves a PDF. Network-dependent.
  - Run:
    ```bash
    cd examples/url_to_pdf
    go run main.go -url https://example.com -o page.pdf
    ```

- Images and Styles
  - Directory: `examples/images_and_styles/`
  - Demonstrates `ConvertFile` with local CSS and images resolved via `Options.ResourcePaths`.
  - Run:
    ```bash
    cd examples/images_and_styles
    go run main.go
    ```

- Simple Invoice
  - Directory: `examples/invoice/`
  - Renders an invoice from a Go HTML template and converts it using `ConvertFile`. Includes basic arithmetic in the template via a custom `mul` function.
  - Run:
    ```bash
    cd examples/invoice
    go run main.go
    ```

- User Report (Tailwind-inspired)
  - Directory: `examples/user_report/`
  - Generates a multi-page, table-heavy report using a Go template and converts it via `ConvertFile`. Configured for US Letter and 36pt margins.
  - Run:
    ```bash
    cd examples/user_report
    go run main.go
    ```
