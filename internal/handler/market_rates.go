package handler

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/investK-tools/investment-calculator-api/internal/service"
)

// GetMarketRates returns Selic, derived CDI, and IPCA 12m from brapi.
func GetMarketRates(api *service.CachedBrAPI) echo.HandlerFunc {
	return func(c *echo.Context) error {
		rates, err := api.FetchMarketRates()
		if err != nil {
			log.Println("failed to fetch market rates:", err)
			return c.String(http.StatusBadGateway, "failed to fetch upstream")
		}
		return c.JSON(http.StatusOK, rates)
	}
}
