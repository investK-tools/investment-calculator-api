package service

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/investK-tools/investment-calculator-api/internal/model"
)

// BrAPIClient encapsulates HTTP interactions with the brapi.dev service.
type BrAPIClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewBrAPIClient constructs a configured BrAPIClient.
func NewBrAPIClient(apiKey string) *BrAPIClient {
	return &BrAPIClient{
		BaseURL:    "https://brapi.dev/api",
		APIKey:     apiKey,
		HTTPClient: http.DefaultClient,
	}
}

// doGet performs a GET to the given path with provided query parameters.
// When APIKey is set, sends Authorization: Bearer <token> on every request (brapi convention).
func (b *BrAPIClient) doGet(path string, query url.Values) (int, string, []byte, error) {
	u, err := url.Parse(fmt.Sprintf("%s%s", b.BaseURL, path))
	if err != nil {
		return 0, "", nil, err
	}

	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return 0, "", nil, err
	}
	req.Header.Set("Accept", "application/json")
	if b.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.APIKey)
	}

	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		return 0, "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	return resp.StatusCode, contentType, body, nil
}

// FetchStockList retrieves and unmarshals the stock list endpoint.
func (b *BrAPIClient) FetchStockList() (model.Response, error) {
	_, _, body, err := b.doGet("/quote/list", nil)
	if err != nil {
		return model.Response{}, err
	}

	var resp model.Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return model.Response{}, err
	}
	return resp, nil
}

// FetchStocksByIDs proxies the /quote/:ids endpoint and accepts incoming query params.
func (b *BrAPIClient) FetchStocksByIDs(ids string, incomingQuery url.Values) (int, string, []byte, error) {
	q := url.Values{}
	for k, vals := range incomingQuery {
		for _, v := range vals {
			q.Add(k, v)
		}
	}
	q.Set("dividends", "true")

	return b.doGet("/quote/"+ids, q)
}

type cashDividend struct {
	PaymentDate   string  `json:"paymentDate"`
	Rate          float64 `json:"rate"`
	LastDatePrior string  `json:"lastDatePrior"`
}

type quoteDividendsData struct {
	CashDividends  []cashDividend `json:"cashDividends"`
	StockDividends []any          `json:"stockDividends"`
	Subscriptions  []any          `json:"subscriptions"`
}

type quoteDividendResult struct {
	Symbol             string              `json:"symbol"`
	RegularMarketPrice float64             `json:"regularMarketPrice"`
	DividendsData      *quoteDividendsData `json:"dividendsData"`
}

type quoteDividendsEnvelope struct {
	Results []quoteDividendResult `json:"results"`
}

// FetchDividendsByIDs calls GET /quote/:ids?dividends=true and maps rows to the legacy parsed.json contract.
func (b *BrAPIClient) FetchDividendsByIDs(ids string, incomingQuery url.Values) (map[string][]model.LegacyDividendRow, error) {
	status, _, body, err := b.FetchStocksByIDs(ids, incomingQuery)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("brapi quote dividends: status %d", status)
	}
	var env quoteDividendsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}
	out := make(map[string][]model.LegacyDividendRow, len(env.Results))
	for _, r := range env.Results {
		key := strings.ToUpper(strings.TrimSpace(r.Symbol))
		if key == "" {
			continue
		}
		if r.DividendsData == nil {
			out[key] = []model.LegacyDividendRow{}
			continue
		}
		cd := r.DividendsData.CashDividends
		if cd == nil {
			cd = []cashDividend{}
		}
		price := r.RegularMarketPrice
		rows := make([]model.LegacyDividendRow, 0, len(cd))
		for _, row := range cd {
			pay := formatLegacyDividendDate(row.PaymentDate, loc)
			rec := formatLegacyDividendDate(row.LastDatePrior, loc)
			if rec == "" {
				rec = pay
			}
			div := row.Rate
			pct := 0.0
			if price > 0 {
				pct = math.Round((div/price)*10000) / 100
			}
			rows = append(rows, model.LegacyDividendRow{
				PayDate:    pay,
				RecordDate: rec,
				Price:      price,
				Percent:    pct,
				Dividend:   div,
			})
		}
		out[key] = rows
	}
	return out, nil
}

func formatLegacyDividendDate(iso string, loc *time.Location) string {
	if strings.TrimSpace(iso) == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		t, err = time.Parse(time.RFC3339, iso)
		if err != nil {
			return iso
		}
	}
	return t.In(loc).Format("02.01.2006")
}

// brapi inflation /v2/inflation
type inflationAPIResponse struct {
	Inflation []inflationPoint `json:"inflation"`
}

type inflationPoint struct {
	Date      string `json:"date"`
	Value     string `json:"value"`
	EpochDate int64  `json:"epochDate"`
}

// brapi prime-rate /v2/prime-rate (JSON key "prime-rate")
type primeRateAPIResponse struct {
	PrimeRate []primeRatePoint `json:"prime-rate"`
}

type primeRatePoint struct {
	Date      string `json:"date"`
	Value     string `json:"value"`
	EpochDate int64  `json:"epochDate"`
}

// FetchInflationBrazil returns raw inflation points from brapi /v2/inflation.
func (b *BrAPIClient) FetchInflationBrazil() ([]inflationPoint, error) {
	q := url.Values{}
	q.Set("country", "brazil")
	status, _, body, err := b.doGet("/v2/inflation", q)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("brapi inflation: status %d", status)
	}
	var resp inflationAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Inflation, nil
}

// FetchPrimeRate returns raw prime-rate points from brapi /v2/prime-rate.
func (b *BrAPIClient) FetchPrimeRate() ([]primeRatePoint, error) {
	q := url.Values{}
	q.Set("country", "brazil")
	status, _, body, err := b.doGet("/v2/prime-rate", q)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("brapi prime-rate: status %d", status)
	}
	var resp primeRateAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.PrimeRate, nil
}

const (
	brazilDateLayout     = "02/01/2006"
	cdiDeltaFromSelicPP  = 0.10 // approximation: CDI typically ~10 b.p. below Selic; not official B3
)

// FetchMarketRates loads the latest IPCA point, Selic from brapi (America/Sao_Paulo for dates),
// and derives CDI as Selic minus cdiDeltaFromSelicPP.
func (b *BrAPIClient) FetchMarketRates() (model.MarketRates, error) {
	var zero model.MarketRates
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return zero, err
	}
	now := time.Now().In(loc)

	inflation, err := b.FetchInflationBrazil()
	if err != nil {
		return zero, err
	}
	ipca, err := selectIPCALatest(inflation)
	if err != nil {
		return zero, err
	}

	pr, err := b.FetchPrimeRate()
	if err != nil {
		return zero, err
	}
	selic, err := selectSelicForTodayOrLatest(pr, now)
	if err != nil {
		return zero, err
	}

	return model.MarketRates{
		SelicAnnualPercent: selic,
		CdiAnnualPercent:   selic - cdiDeltaFromSelicPP,
		Ipca12mPercent:     ipca,
	}, nil
}

// selectIPCALatest returns the IPCA value for the most recent inflation point:
// highest epochDate, breaking ties with the latest parsed calendar date.
func selectIPCALatest(points []inflationPoint) (float64, error) {
	if len(points) == 0 {
		return 0, fmt.Errorf("no inflation data")
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}
	var bestEpoch int64
	var bestT time.Time
	var bestVal float64
	var found bool
	for _, p := range points {
		v, err := strconv.ParseFloat(p.Value, 64)
		if err != nil {
			continue
		}
		t, err := time.ParseInLocation(brazilDateLayout, p.Date, loc)
		if err != nil {
			continue
		}
		if !found || p.EpochDate > bestEpoch || (p.EpochDate == bestEpoch && t.After(bestT)) {
			bestEpoch = p.EpochDate
			bestT = t
			bestVal = v
			found = true
		}
	}
	if !found {
		return 0, fmt.Errorf("no valid inflation value")
	}
	return bestVal, nil
}

// selectSelicForTodayOrLatest prefers entries whose date is today; if none, uses the row with max epochDate.
func selectSelicForTodayOrLatest(points []primeRatePoint, now time.Time) (float64, error) {
	if len(points) == 0 {
		return 0, fmt.Errorf("no prime-rate data")
	}
	today := now.Format(brazilDateLayout)
	var todayBestEpoch int64 = -1
	var todayVal float64
	var todayFound bool
	var globalBestEpoch int64 = -1
	var globalVal float64
	var globalOK bool
	for _, p := range points {
		v, err := strconv.ParseFloat(p.Value, 64)
		if err != nil {
			continue
		}
		if !globalOK || p.EpochDate >= globalBestEpoch {
			globalBestEpoch = p.EpochDate
			globalVal = v
			globalOK = true
		}
		if p.Date == today {
			if !todayFound || p.EpochDate >= todayBestEpoch {
				todayBestEpoch = p.EpochDate
				todayVal = v
				todayFound = true
			}
		}
	}
	if todayFound {
		return todayVal, nil
	}
	if !globalOK {
		return 0, fmt.Errorf("no valid prime-rate value")
	}
	return globalVal, nil
}
