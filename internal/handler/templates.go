package handler

import (
	"html/template"
	"time"
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
	}
}
