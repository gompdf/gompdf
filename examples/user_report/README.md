# Tailwind-Styled PDF Report Example

This example demonstrates how to generate a multi-page PDF report with Tailwind-inspired styling using the gompdf library. The report includes a header, summary statistics, a paginated data table, and a footer with page numbers.

## Features

- **Tailwind-inspired styling**: Clean, modern UI using CSS variables and utility classes
- **Multi-page layout**: Automatically paginated content with proper headers and footers
- **Dynamic data**: Generates 100 sample users with randomized data
- **Responsive tables**: Table layout optimized for PDF rendering
- **Page numbering**: Includes page numbers in the footer

## How It Works

1. The example generates sample user data with randomized names, emails, and activity statistics
2. An HTML template is rendered with this data using Go's template package
3. The HTML is converted to PDF using the gompdf library
4. The PDF is saved to disk with proper US Letter page size and 36pt (0.5in) margins

## Running the Example

```bash
cd examples/user_report
go run main.go
```

This will generate two files:
- `report.html`: The intermediate HTML file
- `report.pdf`: The final PDF report

## Code Structure

- `main.go`: Contains the Go code to generate sample data and convert HTML to PDF
- `report.tmpl.html`: The HTML template with Tailwind-inspired CSS styling

## Implementation Notes

- The example uses inline CSS rather than external Tailwind CSS to ensure proper rendering
- Table-based layouts are used for certain sections to ensure compatibility with PDF rendering
- CSS variables provide a consistent color scheme throughout the document
- The example configures US Letter size with 36-point (0.5in) margins

## Customization

You can customize this example by:
- Modifying the CSS variables at the top of the HTML template
- Changing the page size and margins in the Go code
- Adding more sections or visualizations to the report
- Connecting to a real data source instead of using sample data
