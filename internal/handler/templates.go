package handler

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// TemplateFuncs returns a FuncMap with custom template functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"divf": func(a, b float64) float64 {
			return a / b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"year": func() int {
			return time.Now().Year()
		},
		"formatWeight": func(n pgtype.Numeric) string {
			if !n.Valid {
				return ""
			}
			f, err := n.Float64Value()
			if err != nil || !f.Valid {
				return ""
			}
			// Format without unnecessary decimal places
			weightStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f.Float64), "0"), ".")
			return weightStr
		},
	}
}
