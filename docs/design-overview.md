# GomPDF Design Overview

This document provides an overview of the design and architecture of GomPDF, a Go library for converting HTML/CSS documents to PDF.

## Architecture

GomPDF follows a pipeline architecture with several stages:

1. **Parsing**: HTML and CSS parsing
2. **Style Resolution**: Applying CSS styles to the DOM
3. **Layout**: Computing the layout of elements
4. **Pagination**: Breaking content into pages
5. **Rendering**: Rendering the content to PDF

```
┌─────────┐    ┌─────────────┐    ┌────────┐    ┌────────────┐    ┌─────────┐
│ Parsing │ -> │ Style       │ -> │ Layout │ -> │ Pagination │ -> │ Render  │
│         │    │ Resolution  │    │        │    │            │    │ to PDF  │
└─────────┘    └─────────────┘    └────────┘    └────────────┘    └─────────┘
```

## Core Components

### Parser

The parser is responsible for parsing HTML and CSS documents. It uses a combination of custom parsers and third-party libraries to create a Document Object Model (DOM) and a CSS Object Model (CSSOM).

- `internal/parser/html`: HTML parsing
- `internal/parser/css`: CSS parsing

### Style Engine

The style engine applies CSS styles to the DOM, resolving cascading and inheritance rules.

- `internal/style/cascade.go`: CSS cascade implementation

### Layout Engine

The layout engine computes the position and size of each element in the document.

- `internal/layout/block.go`: Block layout algorithm
- `internal/layout/inline.go`: Inline layout algorithm

### Text Processing

Text processing components handle text shaping, bidirectional text, and font management.

- `internal/text/shaping.go`: Text shaping
- `internal/text/bidi.go`: Bidirectional text support

### Pagination

The pagination component breaks content into pages according to page size and margins.

- `internal/pagination/paginate.go`: Pagination algorithm

### PDF Renderer

The PDF renderer generates the final PDF output.

- `internal/render/pdf/pdf.go`: PDF generation

## API Layer

The API layer provides a simple interface for users to interact with GomPDF.

- `pkg/api/api.go`: Main API
- `pkg/api/options.go`: Configuration options

## Resource Management

The resource management component handles loading and managing external resources like fonts and images.

- `internal/res/loader.go`: Resource loading

## Design Principles

1. **Modularity**: Each component has a clear responsibility and can be developed and tested independently.
2. **Performance**: Efficient algorithms and data structures are used to ensure good performance.
3. **Correctness**: Adherence to web standards for HTML and CSS rendering.
4. **Extensibility**: The architecture allows for easy extension with new features.

## Future Directions

See the [roadmap](roadmap.md) for planned improvements and new features.
