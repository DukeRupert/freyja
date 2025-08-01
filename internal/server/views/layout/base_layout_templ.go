// Code generated by templ - DO NOT EDIT.

// templ: version: v0.3.898
// /internal/backend/templates/layout/base_layout.templ

package layout

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

type BaseLayoutData struct {
	Title       string
	CurrentPage string // "dashboard", "products", "orders", "customers"
	PageTitle   string
	Breadcrumbs []Breadcrumb
}

func BaseLayout(data BaseLayoutData) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 1, "<!doctype html><html class=\"h-full bg-gray-100\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var2 string
		templ_7745c5c3_Var2, templ_7745c5c3_Err = templ.JoinStringErrs(data.Title)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `internal/server/views/layout/base_layout.templ`, Line: 17, Col: 22}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var2))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 2, "</title><link href=\"https://cdn.jsdelivr.net/npm/daisyui@5\" rel=\"stylesheet\" type=\"text/css\"><script src=\"https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4\"></script><script src=\"https://unpkg.com/htmx.org@1.9.12/dist/htmx.min.js\"></script><style>\n\nul {\n        list-style-type: none;\n    }\n\n\t/* Shift In Animation - items slide down and fade in */\n        @keyframes shiftIn {\n            0% {\n                transform: translateY(-20px);\n                opacity: 0;\n                max-height: 0;\n                margin-bottom: 0;\n                padding-top: 0;\n                padding-bottom: 0;\n            }\n            50% {\n                transform: translateY(-10px);\n                opacity: 0.5;\n                max-height: 200px;\n            }\n            100% {\n                transform: translateY(0);\n                opacity: 1;\n                max-height: 200px;\n                margin-bottom: 0.5rem;\n                padding-top: 1rem;\n                padding-bottom: 1rem;\n            }\n        }\n\n        /* Alternative: Slide and scale in */\n        @keyframes shiftInScale {\n            0% {\n                transform: translateY(-30px) scale(0.9);\n                opacity: 0;\n                max-height: 0;\n            }\n            100% {\n                transform: translateY(0) scale(1);\n                opacity: 1;\n                max-height: 200px;\n            }\n        }\n\n        /* Alternative: Elastic shift in */\n        @keyframes shiftInElastic {\n            0% {\n                transform: translateY(-40px);\n                opacity: 0;\n                max-height: 0;\n            }\n            60% {\n                transform: translateY(5px);\n                opacity: 0.8;\n                max-height: 200px;\n            }\n            100% {\n                transform: translateY(0);\n                opacity: 1;\n                max-height: 200px;\n            }\n        }\n\n        .shift-in {\n            animation: shiftIn 0.4s cubic-bezier(0.4, 0, 0.2, 1) forwards;\n        }\n\n        .shift-in-scale {\n            animation: shiftInScale 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;\n        }\n\n        .shift-in-elastic {\n            animation: shiftInElastic 0.6s cubic-bezier(0.68, -0.55, 0.265, 1.55) forwards;\n        }\n\ttr.htmx-swapping { \n\topacity: 0; \n\ttransition: opacity 1s ease-out; \n\t}\n\tli.htmx-swapping { \n\tanimation: shiftIn 0.4s cubic-bezier(0.4, 0, 0.2, 1) forwards;\n\t}\n   @keyframes fade-in {\n     from { opacity: 0; }\n   }\n\n   @keyframes fade-out {\n     to { opacity: 0; }\n   }\n\n   @keyframes slide-from-right {\n     from { transform: translateX(90px); }\n   }\n\n   @keyframes slide-to-left {\n     to { transform: translateX(-90px); }\n   }\n\n   .slide-it {\n     view-transition-name: slide-it;\n   }\n\n   ::view-transition-old(slide-it) {\n     animation: 180ms cubic-bezier(0.4, 0, 1, 1) both fade-out,\n     600ms cubic-bezier(0.4, 0, 0.2, 1) both slide-to-left;\n   }\n   ::view-transition-new(slide-it) {\n     animation: 420ms cubic-bezier(0, 0, 0.2, 1) 90ms both fade-in,\n     600ms cubic-bezier(0.4, 0, 0.2, 1) both slide-from-right;\n   }\n</style></head><body class=\"h-full\"><div class=\"min-h-full\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = Navigation(data.CurrentPage).Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = PageHeader(data.PageTitle, data.Breadcrumbs).Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Var3 := templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
			templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
			templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
			if !templ_7745c5c3_IsBuffer {
				defer func() {
					templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
					if templ_7745c5c3_Err == nil {
						templ_7745c5c3_Err = templ_7745c5c3_BufErr
					}
				}()
			}
			ctx = templ.InitializeContext(ctx)
			templ_7745c5c3_Err = templ_7745c5c3_Var1.Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			return nil
		})
		templ_7745c5c3_Err = MainContent().Render(templ.WithChildren(ctx, templ_7745c5c3_Var3), templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 3, "</div><div id=\"modal\"></div><div id=\"toast\" class=\"fixed bottom-4 left-4 slide-it\"></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = BaseLayoutScripts().Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 4, "</body></html>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

var _ = templruntime.GeneratedTemplate
