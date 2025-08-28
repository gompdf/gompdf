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
	// Create a slice to hold pages
	pages := make([]*Page, 0)
	
	// Function to create a new page
	newPage := func() {
		page := &Page{
			Width:  p.PageSize.Width,
			Height: p.PageSize.Height,
			Boxes:  make([]layout.Box, 0),
		}
		pages = append(pages, page)
	}
	
	// Create first page
	newPage()
	
	// Get the content container
	container := getContentContainer(rootBox)
	if container == nil {
		return pages
	}

	// Collect all content boxes
	var contentBoxes []layout.Box

	// Collect all boxes from the container
	collectBoxes(container, &contentBoxes)
	
	// Sort content boxes by Y position
	sortBoxesByPosition(contentBoxes)

	// Calculate how many pages we need based on content height
	totalHeight := 0.0
	if len(contentBoxes) > 0 {
		totalHeight = contentBoxes[len(contentBoxes)-1].GetY() + contentBoxes[len(contentBoxes)-1].GetHeight() - contentBoxes[0].GetY()
	}
	
	pageHeight := p.PageSize.Height - float64(p.Margins.Top) - float64(p.Margins.Bottom)
	pageCount := int(math.Ceil(totalHeight / pageHeight))
	
	// Ensure we have at least one page
	if pageCount < 1 {
		pageCount = 1
	}
	
	// Create additional pages as needed
	for i := 1; i < pageCount; i++ {
		newPage()
	}

	// Determine which content belongs on which page
	pageBoxes := make(map[int][]layout.Box)
	processedBoxes := make(map[layout.Box]bool)
	
	// First identify headers and footers to keep them on every page
	headerBoxes := make([]layout.Box, 0)
	footerBoxes := make([]layout.Box, 0)
	
	// Identify headers (typically at the top of the document)
	for _, box := range contentBoxes {
		if isHeader(box) {
			headerBoxes = append(headerBoxes, box)
			processedBoxes[box] = true
		}
	}
	
	// Identify footers (typically at the bottom of the document)
	for _, box := range contentBoxes {
		if isFooter(box) {
			footerBoxes = append(footerBoxes, box)
			processedBoxes[box] = true
		}
	}

	// First pass: assign content to pages based on position
	for _, box := range contentBoxes {
		// Skip if we've already processed this box (header/footer)
		if processedBoxes[box] {
			continue
		}
		
		// Mark as processed
		processedBoxes[box] = true
		
		// Calculate box position
		boxY := box.GetY()
		
		// Calculate content area boundaries
		contentStartY := contentBoxes[0].GetY()
		
		// Calculate which page this content belongs to
		relativeY := boxY - contentStartY
		pageIndex := int(math.Floor(relativeY / pageHeight))
		
		// Special case: ensure content at the top of the document stays on first page
		if relativeY < pageHeight * 0.2 { // Top 20% of first page
			pageIndex = 0
		}
		
		// If content is a table row, check if it should be kept with its table
		if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil && blockBox.Node.Data == "tr" {
			// Keep table rows together when possible
			// Logic for keeping table rows together will be handled in the table row mapping
		}
		
		// Ensure we have enough pages
		for pageIndex >= len(pages) {
			newPage()
		}
		
		// Add to page boxes collection
		pageBoxes[pageIndex] = append(pageBoxes[pageIndex], box)
	}

	// Create a map to track which page each table row belongs to
	tableRowPageMap := make(map[string]int) // Maps row ID to page index

	// First pass: identify all table rows and their assigned pages
	// Find the last table row to give it special handling
	var lastTableRow *layout.BlockBox
	var lastTableRowY float64
	
	// First identify the last table row by finding the one with the largest Y value
	for _, box := range contentBoxes {
		if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
			// Check if this is a table row
			if blockBox.Node.Data == "tr" {
				// If this is the first table row we've found or it has a larger Y value
				if lastTableRow == nil || box.GetY() > lastTableRowY {
					lastTableRow = blockBox
					lastTableRowY = box.GetY()
				}
			}
		}
	}
	
	// Now assign all table rows to pages
	for _, box := range contentBoxes {
		if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
			// Check if this is a table row
			if blockBox.Node.Data == "tr" {
				// Get the row ID
				rowID := fmt.Sprintf("%p", blockBox.Node)
				
				// Calculate relative position from content start
				relativePosition := box.GetY() - contentBoxes[0].GetY()
				
				// Determine which page this box belongs to
				desiredIndex := int(math.Floor(relativePosition / pageHeight))
				
				// Ensure we don't exceed page count
				if desiredIndex >= len(pages) {
					desiredIndex = len(pages) - 1
				}
				
				// Special handling for the last table row - ensure it stays with its table
				if lastTableRow != nil {
					rowID := fmt.Sprintf("%p", lastTableRow.Node)
					
					// If this is the last row of a table, we want to keep it with the table
					// by placing it on the same page as the previous row if possible
					if lastTableRow.GetY() - contentBoxes[0].GetY() > pageHeight * float64(len(pages)-1) {
						// This is on the last page, keep it there
						lastRowPage := len(pages) - 1
						tableRowPageMap[rowID] = lastRowPage
					} else {
						// Assign regular rows to their calculated page
						tableRowPageMap[rowID] = desiredIndex
					}
				} else {
					// Assign regular rows to their calculated page
					tableRowPageMap[rowID] = desiredIndex
				}
			}
		}
	}

	distributeContentToPages(pages, pageBoxes, tableRowPageMap, contentBoxes, &p.Margins)

	// Post-processing: remove empty pages
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
	// Get the first content box Y position as reference
	baseY := 0.0
	if len(contentBoxes) > 0 {
		baseY = contentBoxes[0].GetY()
	}
	
	// Calculate the effective page height (content area)
	pageHeight := pages[0].Height - float64(margins.Top) - float64(margins.Bottom)
	
	// Track boxes already added to pages to avoid duplicates
	addedBoxes := make(map[string]bool)
	
	// Place content boxes on their respective pages
	for pageIndex, boxes := range pageBoxes {
		processedRows := make(map[string]bool)
		
		for _, box := range boxes {
			// Generate a unique ID for this box
			boxID := fmt.Sprintf("%p", box)
			
			// Skip if we've already added this box to a page
			if addedBoxes[boxID] {
				continue
			}
			
			// Mark this box as added
			addedBoxes[boxID] = true
			
			// Skip if this is a table row that belongs to another page
			if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil && blockBox.Node.Data == "tr" {
				rowID := fmt.Sprintf("%p", blockBox.Node)
				
				// If this row is assigned to another page or already processed, skip it
				if tableRowPageMap[rowID] != pageIndex || processedRows[rowID] {
					continue
				}
				
				processedRows[rowID] = true
			}
			
			// Clone the box for this page to avoid sharing references
			boxClone := cloneBox(box)
			
			// Calculate new Y position based on page index
			var newY float64
			
			if pageIndex == 0 {
				// For first page, keep original position
				newY = boxClone.GetY()
			} else {
				// For subsequent pages, calculate position relative to page top
				// Calculate which page this content would naturally fall on
				relativeY := boxClone.GetY() - baseY
				naturalPageIndex := int(math.Floor(relativeY / pageHeight))
				
				// If this content is being placed on a different page than its natural page,
				// adjust its position accordingly
				if naturalPageIndex != pageIndex {
					// Content is being moved to a different page
					// Position it at the top of the target page
					newY = float64(margins.Top)
				} else {
					// Content is on its natural page
					// Calculate its position within the page
					positionWithinPage := relativeY - (float64(naturalPageIndex) * pageHeight)
					newY = float64(margins.Top) + positionWithinPage
				}
			}
			
			// Calculate how much to shift the box
			deltaY := newY - boxClone.GetY()
			shiftBox(boxClone, 0, deltaY)
			
			// Add box to the appropriate page
			pages[pageIndex].Boxes = append(pages[pageIndex].Boxes, boxClone)
		}
	}
}

// shiftBox moves a box and all its descendants by (dx, dy)
func shiftBox(box layout.Box, dx, dy float64) {
	box.SetPosition(box.GetX()+dx, box.GetY()+dy)
	switch b := box.(type) {
	case *layout.BlockBox:
		for _, ch := range b.Children {
			shiftBox(ch, dx, dy)
		}
	case *layout.InlineBox:
		for _, ch := range b.Children {
			shiftBox(ch, dx, dy)
		}
	}
}

// getContentContainer returns the main content container (usually body)
func getContentContainer(root layout.Box) layout.Box {
	// If this is already a block box, return it
	if blockBox, ok := root.(*layout.BlockBox); ok {
		return blockBox
	}
	
	// Otherwise, return the original box
	return root
}

// collectBoxes recursively collects all boxes from a container
func collectBoxes(container layout.Box, boxes *[]layout.Box) {
	if container == nil || boxes == nil {
		return
	}

	// Add this box to the collection
	*boxes = append(*boxes, container)

	// Process children based on box type
	switch b := container.(type) {
	case *layout.BlockBox:
		for _, child := range b.Children {
			collectBoxes(child, boxes)
		}
	case *layout.InlineBox:
		// Add all children of inline box
		for _, child := range b.Children {
			*boxes = append(*boxes, child)
			collectBoxes(child, boxes)
		}
	}

}

// sortBoxesByPosition sorts boxes by their Y position using a more efficient algorithm
func sortBoxesByPosition(boxes []layout.Box) {
	// Use Go's built-in sort package with a custom less function
	sort.Slice(boxes, func(i, j int) bool {
		// Primary sort by Y position
		yDiff := boxes[i].GetY() - boxes[j].GetY()
		
		// If Y positions are very close (within 1pt), consider them at the same level
		if math.Abs(yDiff) < 1.0 {
			// Secondary sort by X position for boxes at the same Y level
			return boxes[i].GetX() < boxes[j].GetX()
		}
		
		// Otherwise sort by Y position
		return yDiff < 0
	})
}

// cloneBox creates a deep copy of a box for replication across pages
func cloneBox(box layout.Box) layout.Box {
	switch b := box.(type) {
	case *layout.BlockBox:
		// Create a new block box with the same properties
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

		// Clone children recursively
		for i, child := range b.Children {
			clone.Children[i] = cloneBox(child)
		}

		return clone

	case *layout.InlineBox:
		// Create a new inline box with the same properties
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

		// Clone children recursively
		for i, child := range b.Children {
			clone.Children[i] = cloneBox(child)
		}

		return clone
	}

	// Default case - should not happen with proper box types
	return box
}

// getElementID generates a unique ID for a block box element
func getElementID(box *layout.BlockBox) string {
	if box == nil || box.Node == nil {
		return ""
	}
	
	// Use address of the node as a unique identifier
	return fmt.Sprintf("%p", box.Node)
}

// CalculatePageCount calculates the number of pages needed
func (p *Paginator) CalculatePageCount(rootBox *layout.BlockBox) int {
	pages := p.Paginate(rootBox)
	return len(pages)
}

// isHeader determines if a box is a header element
func isHeader(box layout.Box) bool {
	if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
		// Check if this is a header tag
		if blockBox.Node.Data == "header" {
			return true
		}
		
		// Check for header class
		for _, attr := range blockBox.Node.Attr {
			if attr.Key == "class" && (strings.Contains(attr.Val, "header") || strings.Contains(attr.Val, "page-header")) {
				return true
			}
		}
		
		// Check if it's positioned at the top of the document
		if blockBox.Y < 100 { // Assuming top 100 points could be header
			// Additional check for header-like elements
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
		// Check if this is a footer tag
		if blockBox.Node.Data == "footer" {
			return true
		}
		
		// Check for footer class
		for _, attr := range blockBox.Node.Attr {
			if attr.Key == "class" && (strings.Contains(attr.Val, "footer") || strings.Contains(attr.Val, "page-footer")) {
				return true
			}
		}
	}
	return false
}
