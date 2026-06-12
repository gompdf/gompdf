package gompdf

import (
	"github.com/gompdf/gompdf/pkg/api"
)

type Converter = api.Converter
type Options = api.Options
type Option = api.Option
type PageOrientation = api.PageOrientation

func New() *Converter                           { return api.New() }
func NewWithOptions(options Options) *Converter { return api.NewWithOptions(options) }
func DefaultOptions() Options                   { return api.DefaultOptions() }

var (
	WithPageSize            = api.WithPageSize
	WithMargins             = api.WithMargins
	WithDPI                 = api.WithDPI
	WithDebug               = api.WithDebug
	WithResourcePath        = api.WithResourcePath
	WithFontDirectory       = api.WithFontDirectory
	WithTitle               = api.WithTitle
	WithAuthor              = api.WithAuthor
	WithSubject             = api.WithSubject
	WithKeywords            = api.WithKeywords
	WithUserAgentStylesheet = api.WithUserAgentStylesheet
	WithPageSizeA4          = api.WithPageSizeA4
	WithPageSizeLetter      = api.WithPageSizeLetter
	WithPageSizeLegal       = api.WithPageSizeLegal
	WithPageOrientation     = api.WithPageOrientation
)

const (
	PageSizeA0Width  = api.PageSizeA0Width
	PageSizeA0Height = api.PageSizeA0Height
	PageSizeA1Width  = api.PageSizeA1Width
	PageSizeA1Height = api.PageSizeA1Height
	PageSizeA2Width  = api.PageSizeA2Width
	PageSizeA2Height = api.PageSizeA2Height
	PageSizeA3Width  = api.PageSizeA3Width
	PageSizeA3Height = api.PageSizeA3Height
	PageSizeA4Width  = api.PageSizeA4Width
	PageSizeA4Height = api.PageSizeA4Height
	PageSizeA5Width  = api.PageSizeA5Width
	PageSizeA5Height = api.PageSizeA5Height
	PageSizeA6Width  = api.PageSizeA6Width
	PageSizeA6Height = api.PageSizeA6Height

	PageSizeLetterWidth  = api.PageSizeLetterWidth
	PageSizeLetterHeight = api.PageSizeLetterHeight
	PageSizeLegalWidth   = api.PageSizeLegalWidth
	PageSizeLegalHeight  = api.PageSizeLegalHeight

	PageOrientationPortrait  = api.PageOrientationPortrait
	PageOrientationLandscape = api.PageOrientationLandscape
)
