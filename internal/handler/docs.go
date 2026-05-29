package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/swaggo/swag"
)

const scalarHTML = `<!doctype html>
<html>
<head>
  <title>Money Tracker API</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <style>body { margin: 0; }</style>
</head>
<body>
  <script id="api-reference" data-url="/api/docs/swagger.json"></script>
  <script src="https://unpkg.com/@scalar/api-reference@1.25.72/dist/browser/standalone.js"></script>
</body>
</html>`

func RegisterDocsRoutes(e *echo.Echo) {
	e.GET("/docs", func(c echo.Context) error {
		c.Response().Header().Set("Content-Security-Policy",
			"default-src 'none'; script-src 'unsafe-inline' https://unpkg.com; style-src 'unsafe-inline' https://unpkg.com; connect-src 'self' https://unpkg.com; font-src https://unpkg.com https://fonts.gstatic.com data:; img-src 'self' data: https:")
		return c.HTML(http.StatusOK, scalarHTML)
	})
	e.GET("/api/docs/swagger.json", func(c echo.Context) error {
		doc, err := swag.ReadDoc()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSONBlob(http.StatusOK, []byte(doc))
	})
}
