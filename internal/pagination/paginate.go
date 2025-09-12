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

// shiftSubtree shifts all descendants of a box by (dx, dy).
// It preserves the relative structure while updating absolute positions.
func shiftSubtree(b layout.Box, dx, dy float64) {
	if dx == 0 && dy == 0 || b == nil {
		return
	}
	switch bb := b.(type) {
	case *layout.BlockBox:
		for _, ch := range bb.Children {
			if ch == nil {
				continue
			}
			ch.SetPosition(ch.GetX()+dx, ch.GetY()+dy)
			shiftSubtree(ch, dx, dy)
		}
	case *layout.InlineBox:
		for _, ch := range bb.Children {
			if ch == nil {
				continue
			}
			ch.SetPosition(ch.GetX()+dx, ch.GetY()+dy)
			shiftSubtree(ch, dx, dy)
		}
	case *layout.ImageBox:
		// ImageBox has no children; nothing further to shift
	}
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
	// Collect only the descendants of the content container, not the container itself,
	// to avoid duplicating the entire subtree on the first page
	if bb, ok := container.(*layout.BlockBox); ok {
		for _, child := range bb.Children {
			collectBoxes(child, &contentBoxes)
		}
	} else {
		// Fallback for non-block containers
		collectBoxes(container, &contentBoxes)
	}
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

	pages = p.reflowByBottomThreshold(pages)

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

	// Helper to collect all node pointers under a box (including itself)
	var gatherNodeKeys func(layout.Box, map[string]struct{})
	gatherNodeKeys = func(x layout.Box, set map[string]struct{}) {
		if x == nil {
			return
		}
		if n := x.GetNode(); n != nil {
			set[fmt.Sprintf("%p", n)] = struct{}{}
		}
		switch bb := x.(type) {
		case *layout.BlockBox:
			for _, ch := range bb.Children {
				gatherNodeKeys(ch, set)
			}
		case *layout.InlineBox:
			for _, ch := range bb.Children {
				gatherNodeKeys(ch, set)
			}
		}
	}

	for _, box := range contentBoxes {
		if addedBoxes[box] {
			continue
		}
	}

	for pageIndex, boxes := range pageBoxes {
		// Ensure we have enough pages to cover referenced indices
		for pageIndex >= len(pages) {
			pages = append(pages, &Page{
				Width:  pages[0].Width,
				Height: pages[0].Height,
				Boxes:  make([]layout.Box, 0),
			})
		}

		// Base metrics (all pages share size)
		basePage := pages[0]
		effectivePageHeight := basePage.Height - margins.Top - margins.Bottom

		// Compute min Y among boxes on the first page to normalize positions
		var firstPageMinY float64
		if pageIndex == 0 {
			firstPageMinY = math.Inf(1)
			for _, b := range boxes {
				if y := b.GetY(); y < firstPageMinY {
					firstPageMinY = y
				}
			}
			if math.IsInf(firstPageMinY, 1) {
				firstPageMinY = 0
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
				id := fmt.Sprintf("%p", blockBox.Node)
				if mappedPage, exists := tableRowPageMap[id]; exists && mappedPage != pageIndex {
					continue
				}
			}

			clonedBox := cloneBox(box)

			// Decide final page and Y position
			targetPageIndex := pageIndex
			if targetPageIndex > 0 {
				relativeY := box.GetY() - contentBoxes[0].GetY()
				positionInPage := relativeY - (float64(targetPageIndex) * effectivePageHeight)
				newY := margins.Top + positionInPage

				// If it overflows, advance pages until it fits
				for newY+clonedBox.GetHeight() > basePage.Height-margins.Bottom {
					targetPageIndex++
					for targetPageIndex >= len(pages) {
						pages = append(pages, &Page{
							Width:  basePage.Width,
							Height: basePage.Height,
							Boxes:  make([]layout.Box, 0),
						})
					}
					positionInPage = relativeY - (float64(targetPageIndex) * effectivePageHeight)
					newY = margins.Top + positionInPage
				}
				if newY < margins.Top {
					newY = margins.Top
				}
				oldX, oldY := clonedBox.GetX(), clonedBox.GetY()
				clonedBox.SetPosition(clonedBox.GetX(), newY)
				shiftSubtree(clonedBox, clonedBox.GetX()-oldX, newY-oldY)
			} else {
				// pageIndex == 0
				normalizedY := margins.Top + (box.GetY() - firstPageMinY)
				if normalizedY < margins.Top {
					normalizedY = margins.Top
				}
				// Check overflow using normalized Y against the page's drawable bottom
				if normalizedY+clonedBox.GetHeight() >= (basePage.Height - margins.Bottom - 0.01) {
					// Move to next page top if it doesn't fit on first page
					targetPageIndex = 1
					for targetPageIndex >= len(pages) {
						pages = append(pages, &Page{
							Width:  basePage.Width,
							Height: basePage.Height,
							Boxes:  make([]layout.Box, 0),
						})
					}
					oldX, oldY := clonedBox.GetX(), clonedBox.GetY()
					clonedBox.SetPosition(clonedBox.GetX(), margins.Top)
					shiftSubtree(clonedBox, clonedBox.GetX()-oldX, margins.Top-oldY)
					// Also remove any boxes already added to page 0 that belong to this box's subtree
					if len(pages[0].Boxes) > 0 {
						subtree := make(map[string]struct{})
						gatherNodeKeys(box, subtree)
						filtered := pages[0].Boxes[:0]
						for _, pb := range pages[0].Boxes {
							keep := true
							if n := pb.GetNode(); n != nil {
								if _, exists := subtree[fmt.Sprintf("%p", n)]; exists {
									keep = false
								}
							}
							if keep {
								filtered = append(filtered, pb)
							}
						}
						pages[0].Boxes = filtered
					}
				} else {
					// Keep on first page at normalized position respecting top margin
					oldX, oldY := clonedBox.GetX(), clonedBox.GetY()
					clonedBox.SetPosition(clonedBox.GetX(), normalizedY)
					shiftSubtree(clonedBox, clonedBox.GetX()-oldX, normalizedY-oldY)
				}
			}

			pages[targetPageIndex].Boxes = append(pages[targetPageIndex].Boxes, clonedBox)
			addedBoxes[box] = true
		}
	}
}

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
			collectBoxes(child, boxes)
		}
	case *layout.ImageBox:
		// No children to collect
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
	case *layout.ImageBox:
		clone := &layout.ImageBox{
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
			Src:           b.Src,
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

// reflowByBottomThreshold moves boxes that overflow the bottom margin to the next page
func (p *Paginator) reflowByBottomThreshold(pages []*Page) []*Page {
	if len(pages) == 0 {
		return pages
	}

	bottomThreshold := p.PageSize.Height - p.Margins.Bottom
	availablePageHeight := p.PageSize.Height - p.Margins.Top - p.Margins.Bottom
	// Helper to collect all node pointers under a box (including itself)
	var gatherNodeKeys func(layout.Box, map[string]struct{})
	gatherNodeKeys = func(x layout.Box, set map[string]struct{}) {
		if x == nil {
			return
		}
		if n := x.GetNode(); n != nil {
			set[fmt.Sprintf("%p", n)] = struct{}{}
		}
		switch bb := x.(type) {
		case *layout.BlockBox:
			for _, ch := range bb.Children {
				gatherNodeKeys(ch, set)
			}
		case *layout.InlineBox:
			for _, ch := range bb.Children {
				gatherNodeKeys(ch, set)
			}
		}
	}

	// Helper to shift subtree of a box by given offset
	var shiftSubtree func(layout.Box, float64, float64)
	shiftSubtree = func(x layout.Box, dx, dy float64) {
		if x == nil {
			return
		}
		x.SetPosition(x.GetX()+dx, x.GetY()+dy)
		switch bb := x.(type) {
		case *layout.BlockBox:
			for _, ch := range bb.Children {
				shiftSubtree(ch, dx, dy)
			}
		case *layout.InlineBox:
			for _, ch := range bb.Children {
				shiftSubtree(ch, dx, dy)
			}
		case *layout.ImageBox:
			// no children
		}
	}

	maxIterations := 50
	for iter := 0; iter < maxIterations; iter++ {
		movedAny := false
		// Iterate through pages and move overflow boxes forward
		for i := 0; i < len(pages); i++ {
			page := pages[i]
			for j := 0; j < len(page.Boxes); {
				b := page.Boxes[j]
				// If the box can never fit on a page, place it at top and stop moving it
				if b.GetHeight() > availablePageHeight {
					if b.GetY() > p.Margins.Top+0.01 {
						oldY := b.GetY()
						b.SetPosition(b.GetX(), p.Margins.Top)
						shiftSubtree(b, 0, p.Margins.Top-oldY)
						movedAny = true
					}
					j++
					continue
				}
				if b.GetY()+b.GetHeight() > bottomThreshold {
					// Remove from current page, and also remove any of its subtree boxes that may have been added separately
					subtree := make(map[string]struct{})
					gatherNodeKeys(b, subtree)
					// First remove b at index j
					page.Boxes = append(page.Boxes[:j], page.Boxes[j+1:]...)
					// Then filter out any other boxes on this source page that belong to the same subtree
					filtered := page.Boxes[:0]
					for _, pb := range page.Boxes {
						keep := true
						if n := pb.GetNode(); n != nil {
							if _, exists := subtree[fmt.Sprintf("%p", n)]; exists {
								keep = false
							}
						}
						if keep {
							filtered = append(filtered, pb)
						}
					}
					page.Boxes = filtered
					// Reset j to re-check current index after filtering
					j = 0
					// Find a destination page where it fits without pushing others
					dst := i + 1
					tries := 0
					for {
						if dst >= len(pages) {
							pages = append(pages, &Page{Width: p.PageSize.Width, Height: p.PageSize.Height, Boxes: make([]layout.Box, 0)})
						}
						nextPage := pages[dst]
						// Compute the current bottom of content on the destination page
						y := p.Margins.Top
						if len(nextPage.Boxes) > 0 {
							maxBottom := p.Margins.Top
							for _, nb := range nextPage.Boxes {
								if nb.GetY()+nb.GetHeight() > maxBottom {
									maxBottom = nb.GetY() + nb.GetHeight()
								}
							}
							y = maxBottom
						}
						// If it doesn't fit on this page, try the following one
						if y+b.GetHeight() > p.PageSize.Height-p.Margins.Bottom {
							dst++
							tries++
							if tries > 1000 {
								// Safety guard: place at top of current dst and break
								y = p.Margins.Top
								b.SetPosition(b.GetX(), y)
								shiftSubtree(b, 0, y-b.GetY())
								pages[dst].Boxes = append(pages[dst].Boxes, b)
								movedAny = true
								break
							}
							continue
						}
						// Place on destination page at computed Y (no shifting of existing boxes)
						oldY := b.GetY()
						b.SetPosition(b.GetX(), y)
						shiftSubtree(b, 0, y-oldY)
						nextPage.Boxes = append(nextPage.Boxes, b)
						movedAny = true
						break
					}
					continue // do not advance j since we removed the element at j
				}
				j++
			}
			// Keep boxes sorted vertically for stability
			if len(page.Boxes) > 1 {
				sort.Slice(page.Boxes, func(a, b int) bool {
					ya := page.Boxes[a].GetY()
					yb := page.Boxes[b].GetY()
					if math.Abs(ya-yb) < 1.0 {
						return page.Boxes[a].GetX() < page.Boxes[b].GetX()
					}
					return ya < yb
				})
			}
		}
		if !movedAny {
			break
		}
	}
	return pages
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
