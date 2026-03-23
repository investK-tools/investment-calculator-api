package model

// MarketRates is the JSON contract for GET /v1/market-rates.
type MarketRates struct {
	SelicAnnualPercent float64 `json:"selicAnnualPercent"`
	CdiAnnualPercent   float64 `json:"cdiAnnualPercent"`
	Ipca12mPercent     float64 `json:"ipca12mPercent"`
}
