package model

type Stock struct {
	Ticker      string  `json:"stock"`
	Name        string  `json:"name"`
	CloseValue  float64 `json:"close"`
	ChangeValue float64 `json:"change"`
	Logo        string  `json:"logo"`
	Type        string  `json:"type"`
}

type Response struct {
	Stocks []Stock `json:"stocks"`
}
