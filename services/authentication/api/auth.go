package api

import (
	"context"
	"net/http"
	"time"

	"github.com/KingWahid/inventory/backend/services/authentication/service"
	"github.com/KingWahid/inventory/backend/services/authentication/stub"
	"github.com/labstack/echo/v4"
)

const endpointTimeout = 5 * time.Second

// PostApiV1AuthLogin handles POST /api/v1/auth/login.
func (h *ServerHandler) PostApiV1AuthLogin(c echo.Context) error {
	var req stub.LoginRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctxTimeout, cancel := context.WithTimeout(c.Request().Context(), endpointTimeout)
	defer cancel()

	result, err := h.service.Login(ctxTimeout, service.LoginInput{
		Email:    string(req.Email),
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    result.TokenType,
		"expires_in":    result.ExpiresIn,
	})
}

// PostApiV1AuthRegister handles POST /api/v1/auth/register.
func (h *ServerHandler) PostApiV1AuthRegister(c echo.Context) error {
	var req stub.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctxTimeout, cancel := context.WithTimeout(c.Request().Context(), endpointTimeout)
	defer cancel()

	result, err := h.service.RegisterTenantAdmin(ctxTimeout, service.RegisterInput{
		TenantName: req.TenantName,
		AdminName:  req.AdminName,
		AdminEmail: string(req.AdminEmail),
		Password:   req.Password,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"tenant_id": result.TenantID,
		"user_id":   result.UserID,
		"email":     result.Email,
	})
}
