package main

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/investK-tools/investment-calculator-api/internal/handler"
	"github.com/investK-tools/investment-calculator-api/internal/service"
)

func main() {
	server := echo.New()

	server.Use(middleware.RequestLogger())
	// Enable CORS for the frontend at https://investment-calculator-alpha-rust.vercel.app.
	// Include localhost only in development environment (ENV or GO_ENV == "development").
	origins := []string{"https://www.simuladorinvestimentos.com.br"}
	env := os.Getenv("ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "development" {
		origins = append(origins, "http://localhost:3000")
	}
	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     origins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	routes := server.Group("/v1")

	// In-memory BRAPI cache; set CACHE_TTL_STOCKS, CACHE_TTL_DIVIDENDS, or CACHE_TTL_MARKET_RATES to "0" to disable per category.
	brapi := service.NewCachedBrAPI(
		service.NewBrAPIClient(os.Getenv("BRAPI_API_KEY")),
		service.ParseCacheTTL("CACHE_TTL_STOCKS", service.DefaultTTLStocks),
		service.ParseCacheTTL("CACHE_TTL_DIVIDENDS", service.DefaultTTLDividends),
		service.ParseCacheTTL("CACHE_TTL_MARKET_RATES", service.DefaultTTLMarketRates),
	)

	routes.GET("/stocks", handler.ListStocks(brapi))
	routes.GET("/market-rates", handler.GetMarketRates(brapi))
	routes.GET("/dividends/:ids", handler.GetDividendsByIDsHandler(brapi))

	port := os.Getenv("PORT")
	if port == "" {
		port = "1323"
	}
	addr := ":" + port
	if err := server.Start(addr); err != nil {
		server.Logger.Error("failed to start server", "error", err)
	}
}
