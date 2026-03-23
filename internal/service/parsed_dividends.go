package service

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/investK-tools/investment-calculator-api/internal/model"
)

// DefaultParsedDividendsPath is the on-disk fallback when brapi cashDividends is empty.
const DefaultParsedDividendsPath = "parsed.json"

// LoadParsedDividendsLegacy reads the legacy JSON map ticker → []LegacyDividendRow.
func LoadParsedDividendsLegacy(path string) (map[string][]model.LegacyDividendRow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string][]model.LegacyDividendRow
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// PickParsedDividends returns rows for a ticker using exact (uppercase) or case-insensitive key match.
func PickParsedDividends(m map[string][]model.LegacyDividendRow, ticker string) ([]model.LegacyDividendRow, bool) {
	want := strings.ToUpper(strings.TrimSpace(ticker))
	if want == "" {
		return nil, false
	}
	if rows, ok := m[want]; ok && len(rows) > 0 {
		return rows, true
	}
	for k, v := range m {
		if strings.EqualFold(k, want) && len(v) > 0 {
			return v, true
		}
	}
	return nil, false
}
