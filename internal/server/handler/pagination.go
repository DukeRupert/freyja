package handler

import (
	"math"

	"github.com/dukerupert/freyja/internal/server/views/component"
)

// CalculatePagination generates pagination data from limit, offset, and total count
func CalculatePagination(limit int32, offset int32, total int64) component.PaginationData {
	// Handle edge cases
	if limit <= 0 {
		limit = 10 // default page size
	}
	if total <= 0 {
		return component.PaginationData{
			CurrentPage: 1,
			Total:       0,
			Start:       0,
			End:         0,
			Pages:       []int{},
		}
	}

	// Calculate basic pagination values
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	currentPage := int(offset/limit) + 1

	// Ensure current page is within valid range
	if currentPage < 1 {
		currentPage = 1
	}
	if currentPage > totalPages {
		currentPage = totalPages
	}

	// Calculate start and end positions (1-based for display)
	start := int(offset) + 1
	end := int(offset) + int(limit)
	if end > int(total) {
		end = int(total)
	}
	if start > int(total) {
		start = int(total)
	}

	// Calculate previous/next pages
	hasPrevious := currentPage > 1
	hasNext := currentPage < totalPages
	
	previousPage := 0
	if hasPrevious {
		previousPage = currentPage - 1
	}
	
	nextPage := 0
	if hasNext {
		nextPage = currentPage + 1
	}

	// Generate page numbers for pagination controls
	pages := generatePageNumbers(currentPage, totalPages)

	return component.PaginationData{
		CurrentPage:  currentPage,
		Total:        int(total),
		Start:        start,
		End:          end,
		HasPrevious:  hasPrevious,
		HasNext:      hasNext,
		PreviousPage: previousPage,
		NextPage:     nextPage,
		Pages:        pages,
	}
}

// generatePageNumbers creates a slice of page numbers for pagination controls
// Shows up to 7 pages with current page in the middle when possible
func generatePageNumbers(currentPage, totalPages int) []int {
	if totalPages <= 7 {
		// Show all pages if 7 or fewer
		pages := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			pages[i] = i + 1
		}
		return pages
	}

	var pages []int
	
	// Always include first page
	pages = append(pages, 1)
	
	// Calculate range around current page
	start := currentPage - 2
	end := currentPage + 2
	
	// Adjust range if too close to beginning
	if start <= 2 {
		start = 2
		end = 6
	}
	
	// Adjust range if too close to end
	if end >= totalPages {
		end = totalPages - 1
		start = totalPages - 5
		if start <= 2 {
			start = 2
		}
	}
	
	// Add ellipsis if there's a gap after first page
	if start > 2 {
		pages = append(pages, -1) // -1 represents ellipsis
	}
	
	// Add middle pages
	for i := start; i <= end; i++ {
		if i > 1 && i < totalPages {
			pages = append(pages, i)
		}
	}
	
	// Add ellipsis if there's a gap before last page
	if end < totalPages-1 {
		pages = append(pages, -1) // -1 represents ellipsis
	}
	
	// Always include last page (if more than 1 page)
	if totalPages > 1 {
		pages = append(pages, totalPages)
	}
	
	return pages
}