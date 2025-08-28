package pagination

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/gompdf/gompdf/internal/layout"
)

// Page represents a single page in the document
type Page struct {
	Width  float64
	Height float64
	Boxes  []layout.Box
}

// PageSize represents standard page sizes
type PageSize struct {
	Width  float64
	Height float64
	Name   string
}

// Standard page sizes in points (1/72 inch)
var (
	PageSizeA4     = PageSize{Width: 595.28, Height: 841.89, Name: "A4"}
	PageSizeLetter = PageSize{Width: 612.00, Height: 792.00, Name: "Letter"}
	PageSizeLegal  = PageSize{Width: 612.00, Height: 1008.00, Name: "Legal"}
	PageSizeA3     = PageSize{Width: 841.89, Height: 1190.55, Name: "A3"}
	PageSizeA5     = PageSize{Width: 419.53, Height: 595.28, Name: "A5"}
)

// Margins represents page margins
type Margins struct {
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

// Paginator handles breaking content into pages
type Paginator struct {
	PageSize PageSize
	Margins  Margins
}

// NewPaginator creates a new paginator
func NewPaginator(pageSize PageSize, margins Margins) *Paginator {
	return &Paginator{
		PageSize: pageSize,
		Margins:  margins,
	}
}

// Paginate creates pages for the PDF by distributing content boxes to pages
func (p *Paginator) Paginate(rootBox layout.Box) []*Page {
	pages := make([]*Page, 0)

	newPage := func() {
		page := &Page{
			Width:  p.PageSize.Width,
			Height: p.PageSize.Height,
			Boxes:  make([]layout.Box, 0),
		}
		pages = append(pages, page)
	}

	newPage()

	container := getContentContainer(rootBox)
	if container == nil {
		return pages
	}
	var contentBoxes []layout.Box
	collectBoxes(container, &contentBoxes)
	sortBoxesByPosition(contentBoxes)

	totalHeight := 0.0
	if len(contentBoxes) > 0 {
		totalHeight = contentBoxes[len(contentBoxes)-1].GetY() + contentBoxes[len(contentBoxes)-1].GetHeight() - contentBoxes[0].GetY()
	}

	pageHeight := p.PageSize.Height - float64(p.Margins.Top) - float64(p.Margins.Bottom)
	pageCount := int(math.Ceil(totalHeight / pageHeight))
	if pageCount < 1 {
		pageCount = 1
	}
	for i := 1; i < pageCount; i++ {
		newPage()
	}

	pageBoxes := make(map[int][]layout.Box)
	processedBoxes := make(map[layout.Box]bool)

	for _, box := range contentBoxes {
		if isHeader(box) {
			processedBoxes[box] = true
		}
	}

	for _, box := range contentBoxes {
		if isFooter(box) {
			processedBoxes[box] = true
		}
	}

	for _, box := range contentBoxes {
		if processedBoxes[box] {
			continue
		}
		processedBoxes[box] = true
		boxY := box.GetY()
		contentStartY := contentBoxes[0].GetY()
		relativeY := boxY - contentStartY
		pageIndex := int(math.Floor(relativeY / pageHeight))
		if relativeY < pageHeight*0.2 {
			pageIndex = 0
		}

		if pageIndex > 0 && math.Mod(relativeY, pageHeight) > (pageHeight*0.9) {
			if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
				if blockBox.Node.Data == "tr" || blockBox.Node.Data == "td" {
					pageIndex = int(math.Floor(relativeY / pageHeight))
				}
			}
		}

		for pageIndex >= len(pages) {
			newPage()
		}

		pageBoxes[pageIndex] = append(pageBoxes[pageIndex], box)
	}
	tableRowPageMap := make(map[string]int) // Maps row ID to page index

	var lastTableRow *layout.BlockBox
	var lastTableRowY float64

	for _, box := range contentBoxes {
		if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
			if blockBox.Node.Data == "tr" {
				if lastTableRow == nil || box.GetY() > lastTableRowY {
					lastTableRow = blockBox
					lastTableRowY = box.GetY()
				}
			}
		}
	}

	for _, box := range contentBoxes {
		if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
			if blockBox.Node.Data == "tr" {
				rowID := fmt.Sprintf("%p", blockBox.Node)
				relativePosition := box.GetY() - contentBoxes[0].GetY()
				desiredIndex := int(math.Floor(relativePosition / pageHeight))

				if desiredIndex >= len(pages) {
					desiredIndex = len(pages) - 1
				}

				if lastTableRow != nil {
					tableRowID := fmt.Sprintf("%p", lastTableRow.Node)

					if lastTableRow.GetY()-contentBoxes[0].GetY() > pageHeight*float64(len(pages)-1) {
						lastRowPage := len(pages) - 1
						tableRowPageMap[tableRowID] = lastRowPage
					} else {
						tableRowPageMap[rowID] = desiredIndex
					}
				} else {
					tableRowPageMap[rowID] = desiredIndex
				}
			}
		}
	}

	distributeContentToPages(pages, pageBoxes, tableRowPageMap, contentBoxes, &p.Margins)

	validPages := make([]*Page, 0, len(pages))
	for _, page := range pages {
		if len(page.Boxes) > 0 {
			validPages = append(validPages, page)
		}
	}
	return validPages
}

// distributeContentToPages places content boxes on their respective pages
func distributeContentToPages(pages []*Page, pageBoxes map[int][]layout.Box, tableRowPageMap map[string]int, contentBoxes []layout.Box, margins *Margins) {
	addedBoxes := make(map[layout.Box]bool)
	contentHashes := make(map[string]bool)

	for _, box := range contentBoxes {
		if addedBoxes[box] {
			continue
		}
	}

	for pageIndex, boxes := range pageBoxes {
		if pageIndex >= len(pages) {
			continue
		}

		page := pages[pageIndex]
		if pageIndex == 0 {
			minY := float64(1000000) // Large initial value
			for _, box := range boxes {
				if box.GetY() < minY {
					minY = box.GetY()
				}
			}
		}

		for _, box := range boxes {
			if addedBoxes[box] {
				continue
			}

			contentHash := ""
			if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
				if blockBox.Node.Data == "tr" || blockBox.Node.Data == "td" {
					contentHash = fmt.Sprintf("%s-%.2f-%.2f", blockBox.Node.Data, box.GetX(), box.GetY())

					if contentHashes[contentHash] {
						continue
					}

					contentHashes[contentHash] = true
				}
			}

			if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil && blockBox.Node.Data == "tr" {
				id := fmt.Sprintf("row-%.2f-%.2f-%s", box.GetX(), box.GetY(), blockBox.Node.Data)
				if mappedPage, exists := tableRowPageMap[id]; exists && mappedPage != pageIndex {
					continue
				}
			}

			clonedBox := cloneBox(box)

			if pageIndex > 0 {
				relativeY := box.GetY() - contentBoxes[0].GetY()
				pageHeight := page.Height - margins.Top - margins.Bottom

				positionInPage := relativeY - (float64(pageIndex) * pageHeight)
				newY := margins.Top + positionInPage

				if newY+clonedBox.GetHeight() > page.Height-margins.Bottom {
					newY = page.Height - margins.Bottom - clonedBox.GetHeight()
				}

				clonedBox.SetPosition(clonedBox.GetX(), newY)
			} else if pageIndex == 0 {
				if clonedBox.GetY()+clonedBox.GetHeight() > page.Height-margins.Bottom {
					continue
				}
			}

			page.Boxes = append(page.Boxes, clonedBox)
			addedBoxes[box] = true
		}
	}
}

// getContentContainer returns the main content container (usually body)
func getContentContainer(root layout.Box) layout.Box {
	if blockBox, ok := root.(*layout.BlockBox); ok {
		return blockBox
	}

	return root
}

// collectBoxes recursively collects all boxes from a container
func collectBoxes(container layout.Box, boxes *[]layout.Box) {
	if container == nil || boxes == nil {
		return
	}

	*boxes = append(*boxes, container)

	switch b := container.(type) {
	case *layout.BlockBox:
		for _, child := range b.Children {
			collectBoxes(child, boxes)
		}
	case *layout.InlineBox:
		for _, child := range b.Children {
			*boxes = append(*boxes, child)
			collectBoxes(child, boxes)
		}
	}

}

// sortBoxesByPosition sorts boxes by their Y position using a more efficient algorithm
func sortBoxesByPosition(boxes []layout.Box) {
	sort.Slice(boxes, func(i, j int) bool {
		yDiff := boxes[i].GetY() - boxes[j].GetY()
		if math.Abs(yDiff) < 1.0 {
			return boxes[i].GetX() < boxes[j].GetX()
		}

		return yDiff < 0
	})
}

// cloneBox creates a deep copy of a box for replication across pages
func cloneBox(box layout.Box) layout.Box {
	switch b := box.(type) {
	case *layout.BlockBox:
		clone := &layout.BlockBox{
			Node:          b.Node,
			Style:         b.Style,
			X:             b.X,
			Y:             b.Y,
			Width:         b.Width,
			Height:        b.Height,
			MarginTop:     b.MarginTop,
			MarginRight:   b.MarginRight,
			MarginBottom:  b.MarginBottom,
			MarginLeft:    b.MarginLeft,
			PaddingTop:    b.PaddingTop,
			PaddingRight:  b.PaddingRight,
			PaddingBottom: b.PaddingBottom,
			PaddingLeft:   b.PaddingLeft,
			BorderTop:     b.BorderTop,
			BorderRight:   b.BorderRight,
			BorderBottom:  b.BorderBottom,
			BorderLeft:    b.BorderLeft,
			Children:      make([]layout.Box, len(b.Children)),
		}

		for i, child := range b.Children {
			clone.Children[i] = cloneBox(child)
		}

		return clone

	case *layout.InlineBox:
		clone := &layout.InlineBox{
			Node:          b.Node,
			Style:         b.Style,
			X:             b.X,
			Y:             b.Y,
			Width:         b.Width,
			Height:        b.Height,
			MarginTop:     b.MarginTop,
			MarginRight:   b.MarginRight,
			MarginBottom:  b.MarginBottom,
			MarginLeft:    b.MarginLeft,
			PaddingTop:    b.PaddingTop,
			PaddingRight:  b.PaddingRight,
			PaddingBottom: b.PaddingBottom,
			PaddingLeft:   b.PaddingLeft,
			BorderTop:     b.BorderTop,
			BorderRight:   b.BorderRight,
			BorderBottom:  b.BorderBottom,
			BorderLeft:    b.BorderLeft,
			Text:          b.Text,
			Children:      make([]layout.Box, len(b.Children)),
		}

		for i, child := range b.Children {
			clone.Children[i] = cloneBox(child)
		}

		return clone
	}

	// Default case - should not happen with proper box types
	return box
}

// CalculatePageCount calculates the number of pages needed
func (p *Paginator) CalculatePageCount(rootBox *layout.BlockBox) int {
	pages := p.Paginate(rootBox)
	return len(pages)
}

// isHeader determines if a box is a header element
func isHeader(box layout.Box) bool {
	if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
		if blockBox.Node.Data == "header" {
			return true
		}

		for _, attr := range blockBox.Node.Attr {
			if attr.Key == "class" && (strings.Contains(attr.Val, "header") || strings.Contains(attr.Val, "page-header")) {
				return true
			}
		}

		if blockBox.Y < 100 {
			if blockBox.Node.Data == "div" || blockBox.Node.Data == "nav" || blockBox.Node.Data == "h1" ||
				blockBox.Node.Data == "h2" || blockBox.Node.Data == "h3" {
				return true
			}
		}
	}
	return false
}

// isFooter determines if a box is a footer element
func isFooter(box layout.Box) bool {
	if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
		if blockBox.Node.Data == "footer" {
			return true
		}

		for _, attr := range blockBox.Node.Attr {
			if attr.Key == "class" && (strings.Contains(attr.Val, "footer") || strings.Contains(attr.Val, "page-footer")) {
				return true
			}
		}
	}
	return false
}
