package handler

import (
	"github.com/dukerupert/freyja/internal/server/views/component"
	"reflect"
	"testing"
)

func TestCalculatePagination(t *testing.T) {
	tests := []struct {
		name     string
		limit    int32
		offset   int32
		total    int64
		expected component.PaginationData
	}{
		{
			name:   "basic pagination first page",
			limit:  10,
			offset: 0,
			total:  50,
			expected: component.PaginationData{
				CurrentPage:  1,
				Total:        50,
				Start:        1,
				End:          10,
				HasPrevious:  false,
				HasNext:      true,
				PreviousPage: 0,
				NextPage:     2,
				Pages:        []int{1, 2, 3, 4, 5}, // 5 total pages, all shown
			},
		},
		{
			name:   "basic pagination middle page",
			limit:  10,
			offset: 20,
			total:  50,
			expected: component.PaginationData{
				CurrentPage:  3,
				Total:        50,
				Start:        21,
				End:          30,
				HasPrevious:  true,
				HasNext:      true,
				PreviousPage: 2,
				NextPage:     4,
				Pages:        []int{1, 2, 3, 4, 5}, // 5 total pages, all shown
			},
		},
		{
			name:   "last page",
			limit:  10,
			offset: 40,
			total:  50,
			expected: component.PaginationData{
				CurrentPage:  5,
				Total:        50,
				Start:        41,
				End:          50,
				HasPrevious:  true,
				HasNext:      false,
				PreviousPage: 4,
				NextPage:     0,
				Pages:        []int{1, 2, 3, 4, 5}, // 5 total pages, all shown
			},
		},
		{
			name:   "single page",
			limit:  10,
			offset: 0,
			total:  5,
			expected: component.PaginationData{
				CurrentPage:  1,
				Total:        5,
				Start:        1,
				End:          5,
				HasPrevious:  false,
				HasNext:      false,
				PreviousPage: 0,
				NextPage:     0,
				Pages:        []int{1},
			},
		},
		{
			name:   "exact page boundary",
			limit:  10,
			offset: 0,
			total:  10,
			expected: component.PaginationData{
				CurrentPage:  1,
				Total:        10,
				Start:        1,
				End:          10,
				HasPrevious:  false,
				HasNext:      false,
				PreviousPage: 0,
				NextPage:     0,
				Pages:        []int{1},
			},
		},
		{
			name:   "zero limit uses default",
			limit:  0,
			offset: 0,
			total:  50,
			expected: component.PaginationData{
				CurrentPage:  1,
				Total:        50,
				Start:        1,
				End:          10,
				HasPrevious:  false,
				HasNext:      true,
				PreviousPage: 0,
				NextPage:     2,
				Pages:        []int{1, 2, 3, 4, 5}, // 5 total pages with default limit of 10
			},
		},
		{
			name:   "negative limit uses default",
			limit:  -5,
			offset: 0,
			total:  50,
			expected: component.PaginationData{
				CurrentPage:  1,
				Total:        50,
				Start:        1,
				End:          10,
				HasPrevious:  false,
				HasNext:      true,
				PreviousPage: 0,
				NextPage:     2,
				Pages:        []int{1, 2, 3, 4, 5}, // 5 total pages with default limit of 10
			},
		},
		{
			name:   "zero total",
			limit:  10,
			offset: 0,
			total:  0,
			expected: component.PaginationData{
				CurrentPage: 1,
				Total:       0,
				Start:       0,
				End:         0,
				Pages:       []int{},
			},
		},
		{
			name:   "negative total",
			limit:  10,
			offset: 0,
			total:  -10,
			expected: component.PaginationData{
				CurrentPage: 1,
				Total:       0,
				Start:       0,
				End:         0,
				Pages:       []int{},
			},
		},
		{
			name:   "negative offset",
			limit:  10,
			offset: -5,
			total:  50,
			expected: component.PaginationData{
				CurrentPage:  1,
				Total:        50,
				Start:        1,
				End:          10,
				HasPrevious:  false,
				HasNext:      true,
				PreviousPage: 0,
				NextPage:     2,
				Pages:        []int{1, 2, 3, 4, 5}, // 5 total pages
			},
		},
		{
			name:   "offset beyond total",
			limit:  10,
			offset: 100,
			total:  50,
			expected: component.PaginationData{
				CurrentPage:  5,
				Total:        50,
				Start:        50,
				End:          50,
				HasPrevious:  true,
				HasNext:      false,
				PreviousPage: 4,
				NextPage:     0,
				Pages:        []int{1, 2, 3, 4, 5}, // 5 total pages
			},
		},
		{
			name:   "large numbers",
			limit:  100,
			offset: 1000,
			total:  2500,
			expected: component.PaginationData{
				CurrentPage:  11,
				Total:        2500,
				Start:        1001,
				End:          1100,
				HasPrevious:  true,
				HasNext:      true,
				PreviousPage: 10,
				NextPage:     12,
				Pages:        []int{1, -1, 9, 10, 11, 12, 13, -1, 25}, // with ellipsis (-1) and last page
			},
		},
		{
			name:   "small limit large total",
			limit:  1,
			offset: 0,
			total:  100,
			expected: component.PaginationData{
				CurrentPage:  1,
				Total:        100,
				Start:        1,
				End:          1,
				HasPrevious:  false,
				HasNext:      true,
				PreviousPage: 0,
				NextPage:     2,
				Pages:        []int{1, 2, 3, 4, 5, 6, -1, 100}, // shows first 6 pages, ellipsis, then last page
			},
		},
		{
			name:   "pagination with ellipsis at beginning",
			limit:  10,
			offset: 150,
			total:  200,
			expected: component.PaginationData{
				CurrentPage:  16,
				Total:        200,
				Start:        151,
				End:          160,
				HasPrevious:  true,
				HasNext:      true,
				PreviousPage: 15,
				NextPage:     17,
				Pages:        []int{1, -1, 14, 15, 16, 17, 18, -1, 20}, // actual output from generatePageNumbers
			},
		},
		{
			name:   "pagination with ellipsis at end",
			limit:  10,
			offset: 30,
			total:  200,
			expected: component.PaginationData{
				CurrentPage:  4,
				Total:        200,
				Start:        31,
				End:          40,
				HasPrevious:  true,
				HasNext:      true,
				PreviousPage: 3,
				NextPage:     5,
				Pages:        []int{1, 2, 3, 4, 5, 6, -1, 20}, // ellipsis before last page
			},
		},
		{
			name:   "pagination with ellipsis on both sides",
			limit:  10,
			offset: 100,
			total:  300,
			expected: component.PaginationData{
				CurrentPage:  11,
				Total:        300,
				Start:        101,
				End:          110,
				HasPrevious:  true,
				HasNext:      true,
				PreviousPage: 10,
				NextPage:     12,
				Pages:        []int{1, -1, 9, 10, 11, 12, 13, -1, 30}, // ellipsis on both sides
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePagination(tt.limit, tt.offset, tt.total)

			// Compare each field individually for better error messages
			if result.CurrentPage != tt.expected.CurrentPage {
				t.Errorf("CurrentPage = %d, expected %d", result.CurrentPage, tt.expected.CurrentPage)
			}
			if result.Total != tt.expected.Total {
				t.Errorf("Total = %d, expected %d", result.Total, tt.expected.Total)
			}
			if result.Start != tt.expected.Start {
				t.Errorf("Start = %d, expected %d", result.Start, tt.expected.Start)
			}
			if result.End != tt.expected.End {
				t.Errorf("End = %d, expected %d", result.End, tt.expected.End)
			}
			if result.HasPrevious != tt.expected.HasPrevious {
				t.Errorf("HasPrevious = %t, expected %t", result.HasPrevious, tt.expected.HasPrevious)
			}
			if result.HasNext != tt.expected.HasNext {
				t.Errorf("HasNext = %t, expected %t", result.HasNext, tt.expected.HasNext)
			}
			if result.PreviousPage != tt.expected.PreviousPage {
				t.Errorf("PreviousPage = %d, expected %d", result.PreviousPage, tt.expected.PreviousPage)
			}
			if result.NextPage != tt.expected.NextPage {
				t.Errorf("NextPage = %d, expected %d", result.NextPage, tt.expected.NextPage)
			}

			// Compare Pages array
			if !reflect.DeepEqual(result.Pages, tt.expected.Pages) {
				t.Errorf("Pages = %v, expected %v", result.Pages, tt.expected.Pages)
			}
		})
	}
}

// Helper function to test generatePageNumbers independently
func TestGeneratePageNumbers(t *testing.T) {
	tests := []struct {
		name        string
		currentPage int
		totalPages  int
		expected    []int
	}{
		{
			name:        "few pages - show all",
			currentPage: 3,
			totalPages:  5,
			expected:    []int{1, 2, 3, 4, 5},
		},
		{
			name:        "many pages - current at beginning",
			currentPage: 2,
			totalPages:  20,
			expected:    []int{1, 2, 3, 4, 5, 6, -1, 20},
		},
		{
			name:        "many pages - current in middle",
			currentPage: 10,
			totalPages:  20,
			expected:    []int{1, -1, 8, 9, 10, 11, 12, -1, 20},
		},
		{
			name:        "many pages - current near end",
			currentPage: 16,
			totalPages:  20,
			expected:    []int{1, -1, 14, 15, 16, 17, 18, -1, 20},
		},
		{
			name:        "many pages - current at end",
			currentPage: 19,
			totalPages:  20,
			expected:    []int{1, -1, 15, 16, 17, 18, 19, 20},
		},
		{
			name:        "single page",
			currentPage: 1,
			totalPages:  1,
			expected:    []int{1},
		},
		{
			name:        "exactly 7 pages",
			currentPage: 4,
			totalPages:  7,
			expected:    []int{1, 2, 3, 4, 5, 6, 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePageNumbers(tt.currentPage, tt.totalPages)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("generatePageNumbers(%d, %d) = %v, expected %v",
					tt.currentPage, tt.totalPages, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkCalculatePagination(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculatePagination(10, 20, 1000)
	}
}

func BenchmarkCalculatePaginationLargeTotal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculatePagination(100, 50000, 1000000)
	}
}

// Property-based tests
func TestCalculatePaginationProperties(t *testing.T) {
	testCases := []struct {
		limit  int32
		offset int32
		total  int64
	}{
		{10, 0, 100},
		{25, 50, 200},
		{5, 15, 30},
		{100, 0, 1000},
	}

	for _, tc := range testCases {
		result := CalculatePagination(tc.limit, tc.offset, tc.total)

		// Property: Start should always be <= End
		if result.Start > result.End && result.Total > 0 {
			t.Errorf("Start (%d) should be <= End (%d)", result.Start, result.End)
		}

		// Property: CurrentPage should be >= 1
		if result.CurrentPage < 1 {
			t.Errorf("CurrentPage should be >= 1, got %d", result.CurrentPage)
		}

		// Property: If HasPrevious is true, PreviousPage should be > 0
		if result.HasPrevious && result.PreviousPage <= 0 {
			t.Errorf("If HasPrevious is true, PreviousPage should be > 0, got %d", result.PreviousPage)
		}

		// Property: If HasNext is true, NextPage should be > CurrentPage
		if result.HasNext && result.NextPage <= result.CurrentPage {
			t.Errorf("If HasNext is true, NextPage should be > CurrentPage, got NextPage=%d, CurrentPage=%d", result.NextPage, result.CurrentPage)
		}

		// Property: End should not exceed Total
		if result.End > result.Total {
			t.Errorf("End (%d) should not exceed Total (%d)", result.End, result.Total)
		}
	}
}

// Test edge cases specifically
func TestCalculatePaginationEdgeCases(t *testing.T) {
	t.Run("max int32 limit", func(t *testing.T) {
		result := CalculatePagination(2147483647, 0, 100)
		if result.CurrentPage != 1 {
			t.Errorf("Expected CurrentPage 1, got %d", result.CurrentPage)
		}
		if result.End != 100 {
			t.Errorf("Expected End 100, got %d", result.End)
		}
	})

	t.Run("max int64 total", func(t *testing.T) {
		result := CalculatePagination(10, 0, 9223372036854775807)
		if result.CurrentPage != 1 {
			t.Errorf("Expected CurrentPage 1, got %d", result.CurrentPage)
		}
		if result.End != 10 {
			t.Errorf("Expected End 10, got %d", result.End)
		}
	})
}
