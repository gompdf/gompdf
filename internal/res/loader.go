package res

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ResourceType represents the type of resource
type ResourceType int

const (
	// ResourceTypeUnknown is an unknown resource type
	ResourceTypeUnknown ResourceType = iota
	// ResourceTypeImage is an image resource
	ResourceTypeImage
	// ResourceTypeFont is a font resource
	ResourceTypeFont
	// ResourceTypeCSS is a CSS resource
	ResourceTypeCSS
	// ResourceTypeOther is any other resource
	ResourceTypeOther
)

// Resource represents a loaded resource
type Resource struct {
	URL      string
	Type     ResourceType
	Data     []byte
	MimeType string
}

// Loader handles loading resources
type Loader struct {
	// Base URL or file path for resolving relative URLs
	BaseURL string

	// Resource cache
	cache     map[string]*Resource
	cacheLock sync.RWMutex

	// Resource search paths
	searchPaths []string

	// HTTP client for remote resources
	client *http.Client
}

// NewLoader creates a new resource loader
func NewLoader(baseURL string) *Loader {
	return &Loader{
		BaseURL:     baseURL,
		cache:       make(map[string]*Resource),
		searchPaths: []string{},
		client:      &http.Client{},
	}
}

// AddSearchPath adds a directory to search for local resources
func (l *Loader) AddSearchPath(path string) {
	l.searchPaths = append(l.searchPaths, path)
}

// Load loads a resource from a URL or file path
func (l *Loader) Load(urlStr string) (*Resource, error) {
	// Check if the resource is already cached
	l.cacheLock.RLock()
	if res, ok := l.cache[urlStr]; ok {
		l.cacheLock.RUnlock()
		return res, nil
	}
	l.cacheLock.RUnlock()

	// Handle data URLs directly
	if strings.HasPrefix(urlStr, "data:") {
		res, err := parseDataURL(urlStr)
		if err != nil {
			return nil, err
		}
		l.cacheLock.Lock()
		l.cache[urlStr] = res
		l.cacheLock.Unlock()
		return res, nil
	}

	resolvedURL, err := l.resolveURL(urlStr)
	if err != nil {
		return nil, err
	}

	var res *Resource
	if strings.HasPrefix(resolvedURL, "http://") || strings.HasPrefix(resolvedURL, "https://") {
		res, err = l.loadRemote(resolvedURL)
	} else {
		res, err = l.loadLocal(resolvedURL)
	}

	if err != nil {
		return nil, err
	}

	l.cacheLock.Lock()
	l.cache[urlStr] = res
	l.cacheLock.Unlock()

	return res, nil
}

// parseDataURL parses a data URL (RFC 2397) and returns a Resource.
// Examples:
//   data:image/png;base64,<base64>
//   data:text/plain,Hello%20World
func parseDataURL(u string) (*Resource, error) {
	// Strip prefix
	if !strings.HasPrefix(u, "data:") {
		return nil, fmt.Errorf("not a data URL")
	}
	s := strings.TrimPrefix(u, "data:")
	// Split metadata and data
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URL")
	}
	meta := parts[0]
	dataPart := parts[1]

	mime := "application/octet-stream"
	isBase64 := false
	if meta == "" {
		// defaults
	} else {
		// meta can be like: image/png;base64 or text/plain;charset=utf-8
		comps := strings.Split(meta, ";")
		if len(comps) > 0 && comps[0] != "" {
			mime = comps[0]
		}
		for _, c := range comps[1:] {
			if strings.EqualFold(strings.TrimSpace(c), "base64") {
				isBase64 = true
			}
		}
	}

	var data []byte
	var err error
	if isBase64 {
		data, err = base64.StdEncoding.DecodeString(dataPart)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 data URL: %w", err)
		}
	} else {
		// The non-base64 form is URL-escaped
		if d, derr := url.QueryUnescape(dataPart); derr == nil {
			data = []byte(d)
		} else {
			data = []byte(dataPart)
		}
	}

	r := &Resource{URL: u, Data: data, MimeType: mime}
	r.Type = determineResourceType(mime, "")
	return r, nil
}

// resolveURL resolves a URL relative to the base URL
func (l *Loader) resolveURL(urlStr string) (string, error) {
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		return urlStr, nil
	}

	if filepath.IsAbs(urlStr) {
		return urlStr, nil
	}

	if !strings.HasPrefix(l.BaseURL, "http://") && !strings.HasPrefix(l.BaseURL, "https://") {
		baseDir := filepath.Dir(l.BaseURL)
		return filepath.Join(baseDir, urlStr), nil
	}

	baseURL, err := url.Parse(l.BaseURL)
	if err != nil {
		return "", err
	}

	relURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	return baseURL.ResolveReference(relURL).String(), nil
}

// loadRemote loads a resource from a remote URL
func (l *Loader) loadRemote(urlStr string) (*Resource, error) {
	resp, err := l.client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := &Resource{
		URL:      urlStr,
		Data:     data,
		MimeType: resp.Header.Get("Content-Type"),
	}

	res.Type = determineResourceType(res.MimeType, urlStr)

	return res, nil
}

// loadLocal loads a resource from a local file
func (l *Loader) loadLocal(path string) (*Resource, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return l.loadFromSearchPaths(path)
		}
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	res := &Resource{
		URL:  path,
		Data: data,
	}

	res.MimeType = determineMimeType(path)

	res.Type = determineResourceType(res.MimeType, path)

	return res, nil
}

// loadFromSearchPaths tries to load a resource from the search paths
func (l *Loader) loadFromSearchPaths(filename string) (*Resource, error) {
	baseFilename := filepath.Base(filename)

	for _, searchPath := range l.searchPaths {
		path := filepath.Join(searchPath, baseFilename)

		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			continue
		}

		res := &Resource{
			URL:  path,
			Data: data,
		}

		res.MimeType = determineMimeType(path)

		res.Type = determineResourceType(res.MimeType, path)

		return res, nil
	}

	return nil, fmt.Errorf("resource not found: %s", filename)
}

// determineMimeType determines the MIME type of a file
func determineMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".svg":
		return "image/svg+xml"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".css":
		return "text/css"
	case ".html", ".htm":
		return "text/html"
	default:
		return "application/octet-stream"
	}
}

// determineResourceType determines the type of a resource
func determineResourceType(mimeType, path string) ResourceType {
	if strings.HasPrefix(mimeType, "image/") {
		return ResourceTypeImage
	}

	if strings.HasPrefix(mimeType, "font/") {
		return ResourceTypeFont
	}

	if mimeType == "text/css" {
		return ResourceTypeCSS
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".tiff", ".tif", ".bmp", ".ico":
		return ResourceTypeImage
	case ".ttf", ".otf", ".woff", ".woff2":
		return ResourceTypeFont
	case ".css":
		return ResourceTypeCSS
	}

	return ResourceTypeOther
}

// LoadImage loads an image resource
func (l *Loader) LoadImage(urlStr string) (*Resource, error) {
	res, err := l.Load(urlStr)
	if err != nil {
		return nil, err
	}

	if res.Type != ResourceTypeImage {
		return nil, fmt.Errorf("resource is not an image: %s", urlStr)
	}

	return res, nil
}

// LoadFont loads a font resource
func (l *Loader) LoadFont(urlStr string) (*Resource, error) {
	res, err := l.Load(urlStr)
	if err != nil {
		return nil, err
	}

	if res.Type != ResourceTypeFont {
		return nil, fmt.Errorf("resource is not a font: %s", urlStr)
	}

	return res, nil
}

// LoadCSS loads a CSS resource
func (l *Loader) LoadCSS(urlStr string) (*Resource, error) {
	res, err := l.Load(urlStr)
	if err != nil {
		return nil, err
	}

	if res.Type != ResourceTypeCSS {
		return nil, fmt.Errorf("resource is not CSS: %s", urlStr)
	}

	return res, nil
}

// LoadHTML loads an HTML resource
func (l *Loader) LoadHTML(urlStr string) (*Resource, error) {
	res, err := l.Load(urlStr)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetReader returns a reader for a resource
func (r *Resource) GetReader() *bytes.Reader {
	return bytes.NewReader(r.Data)
}

// GetString returns the resource data as a string
func (r *Resource) GetString() string {
	return string(r.Data)
}
