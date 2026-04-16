package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func main() {
	cfg := viper.New()
	cfg.AutomaticEnv()
	cfg.SetDefault("APP_PORT", "8082")
	port := cfg.GetString("APP_PORT")

	e := echo.New()
	e.HideBanner = true

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	auth := e.Group("/api/v1/auth")
	auth.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.Logger.Fatal(e.Start(":" + port))
}
