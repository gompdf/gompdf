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
		
		// Special case: ensure content near the bottom of a page is properly assigned
		// to avoid duplicates across page boundaries
		if pageIndex > 0 && math.Mod(relativeY, pageHeight) > (pageHeight * 0.9) {
			// If content is in the bottom 10% of a page, ensure it's consistently assigned
			// to the same page to prevent duplicates
			if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
				// For table rows, ensure they stay on the same page as their siblings
				if blockBox.Node.Data == "tr" || blockBox.Node.Data == "td" {
					// Use a consistent page assignment for table rows near page boundaries
					pageIndex = int(math.Floor(relativeY / pageHeight))
				}
			}
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

// These functions have been replaced with direct formatting in the code

// distributeContentToPages places content boxes on their respective pages
func distributeContentToPages(pages []*Page, pageBoxes map[int][]layout.Box, tableRowPageMap map[string]int, contentBoxes []layout.Box, margins *Margins) {
	// Track which boxes have been added to avoid duplicates
	addedBoxes := make(map[layout.Box]bool)
	// Track boxes by their content hash to avoid duplicates across pages
	contentHashes := make(map[string]bool)
	
	// First, handle headers and footers - they should appear on every page
	for _, box := range contentBoxes {
		// Skip if already processed
		if addedBoxes[box] {
			continue
		}
		
		// Check if this is a header or footer
		isHeaderElement := isHeader(box)
		isFooterElement := isFooter(box)
		
		if isHeaderElement || isFooterElement {
			// Add to all pages
			for i := range pages {
				// Clone the box for each page to avoid shared references
				clonedBox := cloneBox(box)
				
				// Position header at the top of each page (after the top margin)
				if isHeaderElement {
					// For headers, keep the original X but adjust Y to top of page + margins
					clonedBox.SetPosition(clonedBox.GetX(), margins.Top)
				}
				
				// Position footer at the bottom of each page (before the bottom margin)
				if isFooterElement {
					// For footers, keep the original X but adjust Y to bottom of page - margins
					clonedBox.SetPosition(clonedBox.GetX(), pages[i].Height - margins.Bottom - box.GetHeight())
				}
				
				// Add to page
				pages[i].Boxes = append(pages[i].Boxes, clonedBox)
			}
			
			// Mark as processed
			addedBoxes[box] = true
		}
	}
	
	// Now handle regular content
	for pageIndex, boxes := range pageBoxes {
		// Skip if page doesn't exist (should never happen)
		if pageIndex >= len(pages) {
			continue
		}
		
		// Get the page
		page := pages[pageIndex]
		
		// For first page, find the minimum Y position of all content
		if pageIndex == 0 {
			// Find the minimum Y position of all content for reference
			minY := float64(1000000) // Large initial value
			for _, box := range boxes {
				if box.GetY() < minY {
					minY = box.GetY()
				}
			}
			// We don't need to store this in baselineY anymore
		}
		
		// Process boxes for this page
		for _, box := range boxes {
			// Skip if already processed
			if addedBoxes[box] {
				continue
			}
			
			// Generate a content hash for duplicate detection across pages
			contentHash := ""
			if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil {
				// For table rows, create a hash based on content
				if blockBox.Node.Data == "tr" || blockBox.Node.Data == "td" {
					// Create a hash based on the node's data and position
					contentHash = fmt.Sprintf("%s-%.2f-%.2f", blockBox.Node.Data, box.GetX(), box.GetY())
					
					// Skip if we've already processed this content
					if contentHashes[contentHash] {
						continue
					}
					
					// Mark this content as processed
					contentHashes[contentHash] = true
				}
			}
			
			// Skip table rows that have been mapped to a different page
			if blockBox, ok := box.(*layout.BlockBox); ok && blockBox.Node != nil && blockBox.Node.Data == "tr" {
				// Generate a simple ID based on position and content
				id := fmt.Sprintf("row-%.2f-%.2f-%s", box.GetX(), box.GetY(), blockBox.Node.Data)
				if mappedPage, exists := tableRowPageMap[id]; exists && mappedPage != pageIndex {
					// This row has been mapped to a different page, skip it
					continue
				}
			}
			
			// Clone the box to avoid shared references
			clonedBox := cloneBox(box)
			
			// Adjust Y position based on page
			if pageIndex > 0 {
				// For subsequent pages, calculate the relative position
				relativeY := box.GetY() - contentBoxes[0].GetY()
				pageHeight := page.Height - margins.Top - margins.Bottom
				
				// Calculate position within the page
				positionInPage := relativeY - (float64(pageIndex) * pageHeight)
				
				// Set the new Y position
				newY := margins.Top + positionInPage
				
				// Ensure we respect the bottom margin
				if newY + clonedBox.GetHeight() > page.Height - margins.Bottom {
					// This box would extend beyond the bottom margin
					// Adjust its position to respect the margin
					newY = page.Height - margins.Bottom - clonedBox.GetHeight()
				}
				
				clonedBox.SetPosition(clonedBox.GetX(), newY)
			} else if pageIndex == 0 {
				// For first page, ensure content respects bottom margin
				if clonedBox.GetY() + clonedBox.GetHeight() > page.Height - margins.Bottom {
					// This box would extend beyond the bottom margin
					// Check if it's a table row that should be kept with its table
					if blockBox, ok := clonedBox.(*layout.BlockBox); ok && blockBox.Node != nil && 
					   (blockBox.Node.Data == "tr" || blockBox.Node.Data == "td") {
						// Skip this box on this page - it will be handled on the next page
						continue
					}
				}
			}
			
			// Add to page
			page.Boxes = append(page.Boxes, clonedBox)
			
			// Mark as processed
			addedBoxes[box] = true
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
