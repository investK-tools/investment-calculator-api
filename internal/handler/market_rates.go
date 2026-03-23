package handler

import (
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"

	"github.com/investK-tools/investment-calculator-api/internal/service"
)

// GetMarketRates returns Selic, derived CDI, and IPCA 12m from brapi.
func GetMarketRates(c *echo.Context) error {
	client := service.NewBrAPIClient(os.Getenv("BRAPI_API_KEY"))
	rates, err := client.FetchMarketRates()
	if err != nil {
		log.Println("failed to fetch market rates:", err)
		return c.String(http.StatusBadGateway, "failed to fetch upstream")
	}
	return c.JSON(http.StatusOK, rates)
}
