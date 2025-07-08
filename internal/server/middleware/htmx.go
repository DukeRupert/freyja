package middleware

import "github.com/labstack/echo/v4"

func HTMXMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Add HTMX detection to context
            isHTMX := c.Request().Header.Get("HX-Request") == "true"
            c.Set("htmx", isHTMX)
            return next(c)
        }
    }
}