package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type TokenHandler struct {
	tokens *service.TokenService
}

func NewTokenHandler(tokens *service.TokenService) *TokenHandler {
	return &TokenHandler{tokens: tokens}
}

func (h *TokenHandler) List(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	tokens, err := h.tokens.List(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, tokens)
}

func (h *TokenHandler) Create(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Name) == "" {
		return response.ValidationError(c, map[string]string{"name": "Nama token wajib diisi"})
	}
	token, err := h.tokens.Create(c.Request().Context(), userID, req.Name)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, token)
}

func (h *TokenHandler) Delete(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	if err := h.tokens.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return respondError(c, apperror.New(apperror.ErrNotFound, "Token tidak ditemukan"))
	}
	return response.Message(c, http.StatusOK, "Token berhasil dihapus", nil)
}
