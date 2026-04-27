package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PaginatedData struct {
	Items   interface{} `json:"items"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
}

func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

func Message(c echo.Context, code int, message string, data interface{}) error {
	return c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{
		Code:    http.StatusCreated,
		Message: "created",
		Data:    data,
	})
}

func Error(c echo.Context, code int, message string) error {
	return c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

func ValidationError(c echo.Context, errors map[string]string) error {
	return c.JSON(http.StatusUnprocessableEntity, Response{
		Code:    http.StatusUnprocessableEntity,
		Message: "Validation failed",
		Data:    map[string]interface{}{"errors": errors},
	})
}

func Paginated(c echo.Context, items interface{}, total int64, page, perPage int) error {
	return c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: PaginatedData{
			Items:   items,
			Total:   total,
			Page:    page,
			PerPage: perPage,
		},
	})
}
