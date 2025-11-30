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
		// Math functions
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"divf": func(a, b float64) float64 {
			return a / b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},

		// Date/Time functions
		"year": func() int {
			return time.Now().Year()
		},

		// String functions
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},

		// Conditional/Logic functions
		"ternary": func(condition bool, trueVal, falseVal interface{}) interface{} {
			if condition {
				return trueVal
			}
			return falseVal
		},
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" || val == 0 {
				return defaultVal
			}
			return val
		},

		// Collection functions
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},

		// Formatting functions
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
