package api

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gompdf/gompdf/internal/layout"
	"github.com/gompdf/gompdf/internal/pagination"
	"github.com/gompdf/gompdf/internal/parser/css"
	"github.com/gompdf/gompdf/internal/parser/html"
	"github.com/gompdf/gompdf/internal/render/pdf"
	"github.com/gompdf/gompdf/internal/res"
	"github.com/gompdf/gompdf/internal/style"
	xhtml "golang.org/x/net/html"
)

// Converter is the main API for converting HTML to PDF
type Converter struct {
	options Options
	loader  *res.Loader
}

// New creates a new HTML to PDF converter with default options
func New() *Converter {
	return NewWithOptions(DefaultOptions())
}

// NewWithOptions creates a new HTML to PDF converter with the specified options
func NewWithOptions(options Options) *Converter {
	return &Converter{
		options: options,
		loader:  res.NewLoader(""),
	}
}

// Convert converts HTML to PDF and writes the result to the specified writer
func (c *Converter) Convert(htmlContent string, output io.Writer) error {
	tempFile, err := os.CreateTemp("", "gompdf-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	err = c.ConvertToFile(htmlContent, tempFile.Name())
	if err != nil {
		return err
	}

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek temporary file: %w", err)
	}

	_, err = io.Copy(output, tempFile)
	if err != nil {
		return fmt.Errorf("failed to copy PDF to output: %w", err)
	}

	return nil
}

// ConvertToFile converts HTML to PDF and writes the result to the specified file
func (c *Converter) ConvertToFile(htmlContent, outputPath string) error {
	if c.loader == nil {
		c.loader = res.NewLoader("")
	}
	for _, path := range c.options.ResourcePaths {
		c.loader.AddSearchPath(path)
	}

	htmlParser := html.NewParser()
	doc, err := htmlParser.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	cssParser := css.NewParser()
	uaStylesheet, err := cssParser.ParseString(c.options.UserAgentStylesheet)
	if err != nil {
		return fmt.Errorf("failed to parse CSS: %w", err)
	}

	styleEngine := style.NewStyleEngine()
	styleEngine.AddStylesheet(uaStylesheet)

	for _, cssText := range collectDocumentStylesheets(doc.Root, c.loader, c.options.Debug) {
		if sheet, parseErr := cssParser.ParseString(cssText); parseErr == nil {
			styleEngine.AddStylesheet(sheet)
		} else if c.options.Debug {
			fmt.Printf("Failed to parse stylesheet: %v\n", parseErr)
		}
	}
	computedStyles := styleEngine.ComputeStyles(doc) // Compute styles and use the result

	pageWidth := c.options.PageWidth
	pageHeight := c.options.PageHeight

	// Determine orientation code based on user option
	orientationCode := "P"
	switch c.options.PageOrientation {
	case PageOrientationLandscape:
		orientationCode = "L"
		// Always swap dimensions for landscape to ensure width > height
		if pageWidth < pageHeight {
			pageWidth, pageHeight = pageHeight, pageWidth
		}
	case PageOrientationPortrait, "":
		orientationCode = "P"
		// Always swap dimensions for portrait to ensure height > width
		if pageWidth > pageHeight {
			pageWidth, pageHeight = pageHeight, pageWidth
		}
	}

	if c.options.Debug {
		fmt.Printf("Page orientation: %s (%s), dimensions: %.2f x %.2f\n",
			c.options.PageOrientation, orientationCode, pageWidth, pageHeight)
	}

	layout.SetMeasurementOrientation(orientationCode)

	layoutEngine := layout.NewEngine()
	layoutEngine.SetOptions(layout.Options{
		Width:  pageWidth,
		Height: pageHeight,
		DPI:    c.options.DPI,
	})
	layoutEngine.Debug = c.options.Debug

	layoutEngine.SetStyles(computedStyles)
	rootBox := layoutEngine.Layout(doc)

	paginationEngine := pagination.NewEngine()
	paginationEngine.SetOptions(pagination.Options{
		PageWidth:    pageWidth,
		PageHeight:   pageHeight,
		MarginTop:    c.options.MarginTop,
		MarginRight:  c.options.MarginRight,
		MarginBottom: c.options.MarginBottom,
		MarginLeft:   c.options.MarginLeft,
	})
	pages := paginationEngine.Paginate(rootBox)

	renderer := pdf.NewRenderer(c.loader)
	renderer.DPI = c.options.DPI
	renderer.Debug = c.options.Debug
	renderer.RenderBackgrounds = c.options.RenderBackgrounds
	renderer.RenderBorders = c.options.RenderBorders
	renderer.DebugDrawBoxes = c.options.DebugDrawBoxes

	for _, dir := range c.options.FontDirectories {
		renderer.AddFontDirectory(dir)
	}
	renderOptions := pdf.RenderOptions{
		Title:       c.options.Title,
		Author:      c.options.Author,
		Subject:     c.options.Subject,
		Keywords:    c.options.Keywords,
		Creator:     "GomPDF", // Use fixed creator since it's not in Options
		Producer:    "GomPDF",
		Orientation: orientationCode, // Pass the orientation to the renderer
	}

	err = renderer.Render(pages, outputPath, renderOptions)
	if err != nil {
		return fmt.Errorf("failed to render PDF: %w", err)
	}

	return nil
}

// collectDocumentStylesheets walks the HTML node tree in document order and
// returns the concatenated list of author stylesheets (external <link rel="stylesheet">
// and inline <style> blocks) preserving source order. The loader is used to
// resolve and load external stylesheets based on the current BaseURL and search paths.
func collectDocumentStylesheets(n *html.Node, loader *res.Loader, debug bool) []string {
	var styles []string

	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		if cur == nil {
			return
		}

		if cur.Type == xhtml.ElementNode {
			// <link rel="stylesheet" href="...">
			if strings.EqualFold(cur.Data, "link") {
				var rel, href string
				for _, a := range cur.Attr {
					if strings.EqualFold(a.Key, "rel") {
						rel = a.Val
					} else if strings.EqualFold(a.Key, "href") {
						href = a.Val
					}
				}
				if href != "" && strings.Contains(strings.ToLower(rel), "stylesheet") {
					if loader != nil {
						if resrc, err := loader.LoadCSS(href); err == nil {
							if debug {
								fmt.Printf("Loaded external stylesheet: %s\n", href)
							}
							styles = append(styles, resrc.GetString())
						} else if debug {
							fmt.Printf("Failed to load external stylesheet %s: %v\n", href, err)
						}
					}
				}
			}

			// <style>...</style>
			if strings.EqualFold(cur.Data, "style") {
				var b strings.Builder
				for c := cur.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == xhtml.TextNode {
						b.WriteString(c.Data)
						b.WriteString("\n")
					}
				}
				if cssText := strings.TrimSpace(b.String()); cssText != "" {
					styles = append(styles, cssText)
				}
			}
		}

		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)
	return styles
}

// ConvertFile converts an HTML file to PDF and writes the result to the specified file
func (c *Converter) ConvertFile(inputPath, outputPath string) error {
	htmlContent, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read HTML file: %w", err)
	}
	c.loader = res.NewLoader(inputPath)
	for _, path := range c.options.ResourcePaths {
		c.loader.AddSearchPath(path)
	}
	return c.ConvertToFile(string(htmlContent), outputPath)
}

// ConvertURL converts an HTML URL to PDF and writes the result to the specified file
func (c *Converter) ConvertURL(url, outputPath string) error {
	c.loader = res.NewLoader(url)
	for _, path := range c.options.ResourcePaths {
		c.loader.AddSearchPath(path)
	}
	resource, err := c.loader.LoadHTML(url)
	if err != nil {
		return fmt.Errorf("failed to load HTML from URL: %w", err)
	}
	return c.ConvertToFile(resource.GetString(), outputPath)
}

// ConvertBytes converts HTML bytes to PDF bytes
func (c *Converter) ConvertBytes(htmlContent []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := c.Convert(string(htmlContent), &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WithOptions returns a new converter with the specified options
func (c *Converter) WithOptions(options Options) *Converter {
	return NewWithOptions(options)
}

// WithOption returns a new converter with the specified option set
func (c *Converter) WithOption(option Option) *Converter {
	newOptions := c.options
	option(&newOptions)
	return NewWithOptions(newOptions)
}

// AddResourcePath adds a path to search for resources
func (c *Converter) AddResourcePath(path string) *Converter {
	newOptions := c.options
	newOptions.ResourcePaths = append(newOptions.ResourcePaths, path)
	return NewWithOptions(newOptions)
}

// AddFontDirectory adds a directory to search for fonts
func (c *Converter) AddFontDirectory(dir string) *Converter {
	newOptions := c.options
	newOptions.FontDirectories = append(newOptions.FontDirectories, dir)
	return NewWithOptions(newOptions)
}

// SetPageSize sets the page size
func (c *Converter) SetPageSize(width, height float64) *Converter {
	newOptions := c.options
	newOptions.PageWidth = width
	newOptions.PageHeight = height
	return NewWithOptions(newOptions)
}

// SetMargins sets the page margins
func (c *Converter) SetMargins(top, right, bottom, left float64) *Converter {
	newOptions := c.options
	newOptions.MarginTop = top
	newOptions.MarginRight = right
	newOptions.MarginBottom = bottom
	newOptions.MarginLeft = left
	return NewWithOptions(newOptions)
}

// SetDPI sets the DPI
func (c *Converter) SetDPI(dpi float64) *Converter {
	newOptions := c.options
	newOptions.DPI = dpi
	return NewWithOptions(newOptions)
}

// SetDebug sets the debug mode
func (c *Converter) SetDebug(debug bool) *Converter {
	newOptions := c.options
	newOptions.Debug = debug
	return NewWithOptions(newOptions)
}

// SetTitle sets the document title
func (c *Converter) SetTitle(title string) *Converter {
	newOptions := c.options
	newOptions.Title = title
	return NewWithOptions(newOptions)
}

// SetAuthor sets the document author
func (c *Converter) SetAuthor(author string) *Converter {
	newOptions := c.options
	newOptions.Author = author
	return NewWithOptions(newOptions)
}

// SetSubject sets the document subject
func (c *Converter) SetSubject(subject string) *Converter {
	newOptions := c.options
	newOptions.Subject = subject
	return NewWithOptions(newOptions)
}

// SetKeywords sets the document keywords
func (c *Converter) SetKeywords(keywords string) *Converter {
	newOptions := c.options
	newOptions.Keywords = keywords
	return NewWithOptions(newOptions)
}
