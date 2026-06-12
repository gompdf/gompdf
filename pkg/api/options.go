package api

// Options represents configuration options for the HTML to PDF converter
type Options struct {
	// Page dimensions
	PageWidth  float64
	PageHeight float64
	// Page orientation: portrait or landscape
	PageOrientation PageOrientation

	// Page margins
	MarginTop    float64
	MarginRight  float64
	MarginBottom float64
	MarginLeft   float64

	// Rendering options
	DPI   float64
	Debug bool

	// Visual rendering toggles
	// When false, backgrounds will not be painted
	RenderBackgrounds bool
	// When false, borders will not be painted
	RenderBorders bool
	// When true, draw debug box overlays (outlines and placeholder backgrounds/labels)
	DebugDrawBoxes bool

	// Testing options
	UseSampleContent bool

	// Resource paths
	ResourcePaths   []string
	FontDirectories []string

	// Document metadata
	Title    string
	Author   string
	Subject  string
	Keywords string

	// Default stylesheets
	UserAgentStylesheet string
}

// Option is a function that modifies Options
type Option func(*Options)

// PageOrientation represents page orientation
type PageOrientation string

const (
	// PageOrientationPortrait sets the page to portrait orientation
	PageOrientationPortrait PageOrientation = "portrait"
	// PageOrientationLandscape sets the page to landscape orientation
	PageOrientationLandscape PageOrientation = "landscape"
)

// DefaultOptions returns the default options
func DefaultOptions() Options {
	return Options{
		// Default to A4 paper size (595.28 x 841.89 points)
		PageWidth:  595.28,
		PageHeight: 841.89,
		// Default page orientation
		PageOrientation: PageOrientationPortrait,

		// Default margins (1 inch = 72 points)
		MarginTop:    72,
		MarginRight:  72,
		MarginBottom: 72,
		MarginLeft:   72,

		// Default DPI
		DPI: 96,

		// Default debug mode
		Debug: false,

		// Default visual toggles
		RenderBackgrounds: false,
		RenderBorders:     false,
		DebugDrawBoxes:    false,

		// Default resource paths
		ResourcePaths:   []string{},
		FontDirectories: []string{},

		// Default document metadata
		Title:    "",
		Author:   "",
		Subject:  "",
		Keywords: "",

		// Default user agent stylesheet
		UserAgentStylesheet: defaultUserAgentStylesheet,
	}
}

// WithPageSize sets the page size
func WithPageSize(width, height float64) Option {
	return func(o *Options) {
		o.PageWidth = width
		o.PageHeight = height
	}
}

// WithMargins sets the page margins
func WithMargins(top, right, bottom, left float64) Option {
	return func(o *Options) {
		o.MarginTop = top
		o.MarginRight = right
		o.MarginBottom = bottom
		o.MarginLeft = left
	}
}

// WithDPI sets the DPI
func WithDPI(dpi float64) Option {
	return func(o *Options) {
		o.DPI = dpi
	}
}

// WithDebug sets the debug mode
func WithDebug(debug bool) Option {
	return func(o *Options) {
		o.Debug = debug
	}
}

// WithResourcePath adds a path to search for resources
func WithResourcePath(path string) Option {
	return func(o *Options) {
		o.ResourcePaths = append(o.ResourcePaths, path)
	}
}

// WithFontDirectory adds a directory to search for fonts
func WithFontDirectory(dir string) Option {
	return func(o *Options) {
		o.FontDirectories = append(o.FontDirectories, dir)
	}
}

// WithTitle sets the document title
func WithTitle(title string) Option {
	return func(o *Options) {
		o.Title = title
	}
}

// WithAuthor sets the document author
func WithAuthor(author string) Option {
	return func(o *Options) {
		o.Author = author
	}
}

// WithSubject sets the document subject
func WithSubject(subject string) Option {
	return func(o *Options) {
		o.Subject = subject
	}
}

// WithKeywords sets the document keywords
func WithKeywords(keywords string) Option {
	return func(o *Options) {
		o.Keywords = keywords
	}
}

// WithUserAgentStylesheet sets the user agent stylesheet
func WithUserAgentStylesheet(stylesheet string) Option {
	return func(o *Options) {
		o.UserAgentStylesheet = stylesheet
	}
}

// WithPageOrientation sets the page orientation
func WithPageOrientation(orientation PageOrientation) Option {
	return func(o *Options) {
		o.PageOrientation = orientation
	}
}

// Standard page sizes in points (1/72 inch)
const (
	// A series
	PageSizeA0Width  = 2383.94
	PageSizeA0Height = 3370.39
	PageSizeA1Width  = 1683.78
	PageSizeA1Height = 2383.94
	PageSizeA2Width  = 1190.55
	PageSizeA2Height = 1683.78
	PageSizeA3Width  = 841.89
	PageSizeA3Height = 1190.55
	PageSizeA4Width  = 595.28
	PageSizeA4Height = 841.89
	PageSizeA5Width  = 419.53
	PageSizeA5Height = 595.28
	PageSizeA6Width  = 297.64
	PageSizeA6Height = 419.53

	// US Letter and Legal
	PageSizeLetterWidth  = 612
	PageSizeLetterHeight = 792
	PageSizeLegalWidth   = 612
	PageSizeLegalHeight  = 1008
)

// WithPageSizeA4 sets the page size to A4
func WithPageSizeA4() Option {
	return WithPageSize(PageSizeA4Width, PageSizeA4Height)
}

// WithPageSizeLetter sets the page size to US Letter
func WithPageSizeLetter() Option {
	return WithPageSize(PageSizeLetterWidth, PageSizeLetterHeight)
}

// WithPageSizeLegal sets the page size to US Legal
func WithPageSizeLegal() Option {
	return WithPageSize(PageSizeLegalWidth, PageSizeLegalHeight)
}

// Default user agent stylesheet
const defaultUserAgentStylesheet = `
/* Default user agent stylesheet */
html, body {
  margin: 0;
  padding: 0;
  font-family: 'Times New Roman', Times, serif;
  font-size: 16px;
  line-height: 1.5;
  color: #000000;
}

h1 {
  font-size: 2em;
  margin: 0.67em 0;
}

h2 {
  font-size: 1.5em;
  margin: 0.75em 0;
}

h3 {
  font-size: 1.17em;
  margin: 0.83em 0;
}

h4 {
  font-size: 1em;
  margin: 1.12em 0;
}

h5 {
  font-size: 0.83em;
  margin: 1.5em 0;
}

h6 {
  font-size: 0.75em;
  margin: 1.67em 0;
}

p {
  margin: 1em 0;
}

b, strong {
  font-weight: bold;
}

i, em {
  font-style: italic;
}

u {
  text-decoration: underline;
}

a {
  color: #0000EE;
  text-decoration: underline;
}

a:visited {
  color: #551A8B;
}

table {
  border-collapse: collapse;
  border-spacing: 0;
}

th, td {
  padding: 0.2em 0.5em;
  border: 1px solid #000000;
}

ul, ol {
  margin: 1em 0;
  padding-left: 40px;
}

ul {
  list-style-type: disc;
}

ol {
  list-style-type: decimal;
}

li {
  display: list-item;
}

blockquote {
  margin: 1em 40px;
}

pre {
  font-family: monospace;
  white-space: pre;
  margin: 1em 0;
}

code {
  font-family: monospace;
}

hr {
  border: 1px solid #000000;
  margin: 0.5em 0;
}

img {
  max-width: 100%;
  height: auto;
}

@page {
  margin: 0.5in;
}
`
