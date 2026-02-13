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
	client := service.NewBrAPIClient("") // no auth required for list endpoint

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

// GetStocksByIDsHandler proxies requests to the upstream API for the provided ids.
// It forwards incoming query parameters and ensures dividends=true is present.
func GetDividendsByIDsHandler(c *echo.Context) error {
	ids := c.Param("ids")
	if ids == "" {
		return c.String(http.StatusBadRequest, "ids path parameter is required")
	}
	// Read the local parsed.json file and filter its top-level keys by the provided ids.
	data, err := os.ReadFile("parsed.json")
	if err != nil {
		log.Println("failed to read parsed.json:", err)
		return c.String(http.StatusInternalServerError, "failed to read parsed.json")
	}

	// Unmarshal into a map so we can pick keys by id.
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		log.Println("failed to unmarshal parsed.json:", err)
		return c.String(http.StatusInternalServerError, "failed to parse parsed.json")
	}

	filtered := make(map[string]json.RawMessage, len(all))
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if v, ok := all[id]; ok {
			filtered[id] = v
		}
	}

	respBytes, err := json.Marshal(filtered)
	if err != nil {
		log.Println("failed to marshal filtered response:", err)
		return c.String(http.StatusInternalServerError, "failed to build response")
	}

	_ = model.Response{} // keep import usage explicit for clarity
	return c.Blob(http.StatusOK, "application/json", respBytes)
}
