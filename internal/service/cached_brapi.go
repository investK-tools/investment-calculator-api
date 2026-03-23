package service

import (
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"

	"github.com/investK-tools/investment-calculator-api/internal/model"
)

// CachedBrAPI wraps BrAPIClient with in-memory TTL caching and singleflight deduplication.
type CachedBrAPI struct {
	client       *BrAPIClient
	cache        *cache.Cache
	stocksTTL    time.Duration
	dividendsTTL time.Duration
	ratesTTL     time.Duration
	sf           singleflight.Group
}

const (
	cacheKeyStockList   = "quote:list"
	cacheKeyMarketRates = "market_rates"

	// Default cache TTLs (overridable via CACHE_TTL_* env).
	DefaultTTLStocks      = 10 * time.Minute
	DefaultTTLDividends   = 30 * time.Minute
	DefaultTTLMarketRates = 30 * time.Minute
)

// ParseCacheTTL reads duration from env (e.g. CACHE_TTL_STOCKS). Empty uses defaultDur.
// Set to "0" to disable caching for that category.
func ParseCacheTTL(envKey string, defaultDur time.Duration) time.Duration {
	s := strings.TrimSpace(os.Getenv(envKey))
	if s == "" {
		return defaultDur
	}
	if s == "0" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultDur
	}
	return d
}

// NewCachedBrAPI returns a cached wrapper. Pass ttl 0 to disable caching for that category.
// defaultExpiration and cleanupInterval control go-cache's internal sweep (not per-key TTL).
func NewCachedBrAPI(client *BrAPIClient, stocksTTL, dividendsTTL, ratesTTL time.Duration) *CachedBrAPI {
	maxTTL := stocksTTL
	if dividendsTTL > maxTTL {
		maxTTL = dividendsTTL
	}
	if ratesTTL > maxTTL {
		maxTTL = ratesTTL
	}
	if maxTTL <= 0 {
		maxTTL = time.Hour
	}
	return &CachedBrAPI{
		client:       client,
		cache:        cache.New(maxTTL, maxTTL*2),
		stocksTTL:    stocksTTL,
		dividendsTTL: dividendsTTL,
		ratesTTL:     ratesTTL,
	}
}

// FetchStockList returns the stock list, using cache when enabled.
func (w *CachedBrAPI) FetchStockList() (model.Response, error) {
	if w.stocksTTL <= 0 {
		return w.client.FetchStockList()
	}
	if x, ok := w.cache.Get(cacheKeyStockList); ok {
		return x.(model.Response), nil
	}
	v, err, _ := w.sf.Do(cacheKeyStockList, func() (interface{}, error) {
		if x, ok := w.cache.Get(cacheKeyStockList); ok {
			return x.(model.Response), nil
		}
		r, err := w.client.FetchStockList()
		if err != nil {
			return nil, err
		}
		w.cache.Set(cacheKeyStockList, r, w.stocksTTL)
		return r, nil
	})
	if err != nil {
		return model.Response{}, err
	}
	return v.(model.Response), nil
}

// FetchMarketRates returns Selic/CDI/IPCA, using cache when enabled.
func (w *CachedBrAPI) FetchMarketRates() (model.MarketRates, error) {
	if w.ratesTTL <= 0 {
		return w.client.FetchMarketRates()
	}
	if x, ok := w.cache.Get(cacheKeyMarketRates); ok {
		return x.(model.MarketRates), nil
	}
	v, err, _ := w.sf.Do(cacheKeyMarketRates, func() (interface{}, error) {
		if x, ok := w.cache.Get(cacheKeyMarketRates); ok {
			return x.(model.MarketRates), nil
		}
		r, err := w.client.FetchMarketRates()
		if err != nil {
			return nil, err
		}
		w.cache.Set(cacheKeyMarketRates, r, w.ratesTTL)
		return r, nil
	})
	if err != nil {
		return model.MarketRates{}, err
	}
	return v.(model.MarketRates), nil
}

// FetchDividendsByIDs calls the upstream dividends mapping with cache when enabled.
func (w *CachedBrAPI) FetchDividendsByIDs(ids string, incomingQuery url.Values) (map[string][]model.LegacyDividendRow, error) {
	if w.dividendsTTL <= 0 {
		return w.client.FetchDividendsByIDs(ids, incomingQuery)
	}
	key := dividendsCacheKey(ids, incomingQuery)
	if x, ok := w.cache.Get(key); ok {
		return x.(map[string][]model.LegacyDividendRow), nil
	}
	v, err, _ := w.sf.Do(key, func() (interface{}, error) {
		if x, ok := w.cache.Get(key); ok {
			return x.(map[string][]model.LegacyDividendRow), nil
		}
		m, err := w.client.FetchDividendsByIDs(ids, incomingQuery)
		if err != nil {
			return nil, err
		}
		w.cache.Set(key, m, w.dividendsTTL)
		return m, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(map[string][]model.LegacyDividendRow), nil
}

func dividendsCacheKey(ids string, q url.Values) string {
	return "dividends:" + normalizeStockIDs(ids) + ":" + canonicalQueryString(q)
}

func normalizeStockIDs(ids string) string {
	parts := strings.Split(ids, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// canonicalQueryString builds a stable query string for cache keys (sorted keys and values).
func canonicalQueryString(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		vals := append([]string(nil), q[k]...)
		sort.Strings(vals)
		for _, v := range vals {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(v))
		}
	}
	return b.String()
}
