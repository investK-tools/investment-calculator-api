package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

// doGet performs a GET to the given path with provided query parameters and optional auth.
func (b *BrAPIClient) doGet(path string, query url.Values, requireAuth bool) (int, string, []byte, error) {
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
	if requireAuth && b.APIKey != "" {
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
	_, _, body, err := b.doGet("/quote/list", nil, false)
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

	return b.doGet("/quote/"+ids, q, true)
}

