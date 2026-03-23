package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/investK-tools/investment-calculator-api/internal/model"
	"github.com/investK-tools/investment-calculator-api/internal/service"
)

// ListStocks returns the list of available stocks from the upstream API.
func ListStocks(api *service.CachedBrAPI) echo.HandlerFunc {
	return func(c *echo.Context) error {
		stocks, err := api.FetchStockList()
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
}

// GetDividendsByIDsHandler returns dividend rows from brapi GET /quote/:ids?dividends=true,
// shaped as a JSON object keyed by requested ticker (legacy parsed.json fields: pay_date, record_date, price, percent, dividend).
func GetDividendsByIDsHandler(api *service.CachedBrAPI) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ids := c.Param("ids")
		if ids == "" {
			return c.String(http.StatusBadRequest, "ids path parameter is required")
		}

		bySymbol, err := api.FetchDividendsByIDs(ids, c.QueryParams())
		if err != nil {
			log.Println("failed to fetch dividends:", err)
			return c.String(http.StatusBadGateway, "failed to fetch upstream")
		}

		parsedPath := os.Getenv("PARSED_DIVIDENDS_PATH")
		if parsedPath == "" {
			parsedPath = service.DefaultParsedDividendsPath
		}

		filtered := make(map[string]json.RawMessage)
		var parsedCache map[string][]model.LegacyDividendRow
		parsedLoaded := false

		for _, id := range strings.Split(ids, ",") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			key := strings.ToUpper(id)
			divs, ok := bySymbol[key]
			if !ok || len(divs) == 0 {
				if !parsedLoaded {
					parsedLoaded = true
					p, err := service.LoadParsedDividendsLegacy(parsedPath)
					if err != nil {
						log.Println("parsed.json fallback:", err)
					} else {
						parsedCache = p
					}
				}
				if parsedCache != nil {
					if fb, found := service.PickParsedDividends(parsedCache, id); found {
						divs = fb
					}
				}
			}
			if len(divs) == 0 {
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
}
