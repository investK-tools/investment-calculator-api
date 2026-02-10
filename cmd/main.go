package main

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"invest-simulator/internal/handler"
)

func main() {
	server := echo.New()

	server.Use(middleware.RequestLogger())
	// Enable CORS for the frontend running on http://localhost:3000
	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	routes := server.Group("/v1")

	routes.GET("/stocks", handler.ListStocks)
	routes.GET("/dividends/:ids", handler.GetDividendsByIDsHandler)

	if err := server.Start(":1323"); err != nil {
		server.Logger.Error("failed to start server", "error", err)
	}
}
