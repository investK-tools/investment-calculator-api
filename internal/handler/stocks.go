package handler

import (
	"log"
	"net/http"
	"os"
	"encoding/json"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/investK-tools/investment-calculator-api/internal/model"
	"github.com/investK-tools/investment-calculator-api/internal/service"
)

// ListStocks returns the list of available stocks from the upstream API.
func ListStocks(c *echo.Context) error {
	client := service.NewBrAPIClient(os.Getenv("BRAPI_API_KEY"))

	stocks, err := client.FetchStockList()
	if err != nil {
		log.Println("failed to fetch stock list:", err)
		return c.String(http.StatusBadGateway, "failed to fetch upstream")
	}
	// Filter to only include stocks of type "fund"
	filtered := make([]model.Stock, 0, len(stocks.Stocks))
	for _, s := range stocks.Stocks {
		if strings.ToLower(s.Type) == "fund" {
			filtered = append(filtered, s)
		}
	}

	return c.JSON(http.StatusOK, model.Response{Stocks: filtered})
}

// GetDividendsByIDsHandler returns dividend rows from brapi GET /quote/:ids?dividends=true,
// shaped as a JSON object keyed by requested ticker (legacy parsed.json fields: pay_date, record_date, price, percent, dividend).
func GetDividendsByIDsHandler(c *echo.Context) error {
	ids := c.Param("ids")
	if ids == "" {
		return c.String(http.StatusBadRequest, "ids path parameter is required")
	}

	client := service.NewBrAPIClient(os.Getenv("BRAPI_API_KEY"))
	bySymbol, err := client.FetchDividendsByIDs(ids, c.QueryParams())
	if err != nil {
		log.Println("failed to fetch dividends:", err)
		return c.String(http.StatusBadGateway, "failed to fetch upstream")
	}

	filtered := make(map[string]json.RawMessage)
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		key := strings.ToUpper(id)
		divs, ok := bySymbol[key]
		if !ok {
			continue
		}
		raw, err := json.Marshal(divs)
		if err != nil {
			log.Println("failed to marshal dividends for", id, ":", err)
			return c.String(http.StatusInternalServerError, "failed to build response")
		}
		filtered[id] = raw
	}

	respBytes, err := json.Marshal(filtered)
	if err != nil {
		log.Println("failed to marshal filtered response:", err)
		return c.String(http.StatusInternalServerError, "failed to build response")
	}

	_ = model.Response{} // keep import usage explicit for clarity
	return c.Blob(http.StatusOK, "application/json", respBytes)
}
