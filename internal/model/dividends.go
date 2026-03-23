package model

// LegacyDividendRow is the JSON shape for GET /v1/dividends/:ids (historical contract from parsed.json).
type LegacyDividendRow struct {
	PayDate    string  `json:"pay_date"`
	RecordDate string  `json:"record_date"`
	Price      float64 `json:"price"`
	Percent    float64 `json:"percent"`
	Dividend   float64 `json:"dividend"`
}
