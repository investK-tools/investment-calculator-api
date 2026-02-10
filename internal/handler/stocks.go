package handler

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v5"

	"invest-simulator/internal/model"
	"invest-simulator/internal/service"
)

// ListStocks returns the list of available stocks from the upstream API.
func ListStocks(c *echo.Context) error {
	client := service.NewBrAPIClient("") // no auth required for list endpoint

	stocks, err := client.FetchStockList()
	if err != nil {
		log.Println("failed to fetch stock list:", err)
		return c.String(http.StatusBadGateway, "failed to fetch upstream")
	}

	return c.JSON(http.StatusOK, stocks)
}

// GetStocksByIDsHandler proxies requests to the upstream API for the provided ids.
// It forwards incoming query parameters and ensures dividends=true is present.
func GetDividendsByIDsHandler(c *echo.Context) error {
	ids := c.Param("ids")
	if ids == "" {
		return c.String(http.StatusBadRequest, "ids path parameter is required")
	}

	client := service.NewBrAPIClient("2HnxXZLUA5nvif3ukvxcte")
	status, contentType, body, err := client.FetchStocksByIDs(ids, c.QueryParams())
	if err != nil {
		log.Println("failed to fetch stocks by ids:", err)
		return c.String(http.StatusBadGateway, "failed to fetch upstream")
	}

	// Ensure response is returned verbatim to preserve upstream structure and headers.
	_ = model.Response{} // keep import usage explicit for clarity
	return c.Blob(status, contentType, body)
}
