package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type epsConfig struct {
	Username          string
	Password          string
	HashKey           string
	MerchantID        string
	StoreID           string
	Sandbox           bool
	Enabled           bool
	TransactionTypeID int
	AppReturnURL      string
}

type epsEndpoints struct {
	token, initialize, verify string
}

func (c epsConfig) endpoints() epsEndpoints {
	if c.Sandbox {
		return epsEndpoints{
			token:      "https://sandboxpgapi.eps.com.bd/v1/Auth/GetToken",
			initialize: "https://sandboxpgapi.eps.com.bd/v1/EPSEngine/InitializeEPS",
			verify:     "https://sandboxpgapi.eps.com.bd/v1/EPSEngine/CheckMerchantTransactionStatus",
		}
	}
	return epsEndpoints{
		token:      "https://pgapi.eps.com.bd/v1/Auth/GetToken",
		initialize: "https://pgapi.eps.com.bd/v1/EPSEngine/InitializeEPS",
		verify:     "https://pgapi.eps.com.bd/v1/EPSEngine/CheckMerchantTransactionStatus",
	}
}

func (c epsConfig) ready() error {
	if !c.Enabled {
		return fmt.Errorf("EPS is disabled for this project")
	}
	if c.Username == "" || c.Password == "" || c.HashKey == "" || c.MerchantID == "" || c.StoreID == "" {
		return fmt.Errorf("EPS credentials are not set")
	}
	return nil
}

type epsDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type epsClient struct {
	cfg   epsConfig
	http  epsDoer
	mu    sync.Mutex
	token string
	exp   time.Time
}

func newEPSClient(cfg epsConfig) *epsClient {
	return &epsClient{cfg: cfg, http: &http.Client{Timeout: 20 * time.Second}}
}

type epsInitResult struct {
	RedirectURL   string
	TransactionID string
}

type epsVerifyResult struct {
	Status                string
	Amount                float64
	Currency              string
	MerchantTransactionID string
	EPSTransactionID      string
	FinancialEntity       string
	TransactionDate       string
	ErrorMessage          string
}

func (c *epsClient) Initialize(ctx context.Context, args map[string]string) (epsInitResult, error) {
	if err := c.cfg.ready(); err != nil {
		return epsInitResult{}, err
	}
	token, err := c.tokenFor(ctx)
	if err != nil {
		return epsInitResult{}, err
	}
	merchantTxn := args["merchant_transaction_id"]
	amount, _ := parseAmount(args["total_amount"])
	if merchantTxn == "" || amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return epsInitResult{}, fmt.Errorf("EPS initialize failed")
	}
	// Empty customer fields make sandbox Success stick as Cancel.
	// ProductList makes InitializeEPS return a generic 500, so keep the
	// customer block and leave the product list off.
	name := first(args["customer_name"], "EPS Test")
	email := first(args["customer_email"], "test@udbhabon.com")
	phone := first(args["customer_phone"], "01700000000")
	address := first(args["customer_address"], "House 1, Road 1")
	city := first(args["customer_city"], "Dhaka")
	state := first(args["customer_state"], "Dhaka")
	postcode := first(args["customer_postcode"], "1200")
	country := first(args["customer_country"], "BD")
	product := first(args["product_name"], "Plan")
	orderID := first(args["customer_order_id"], "ORD"+merchantTxn)
	body := map[string]interface{}{
		"merchantId":            c.cfg.MerchantID,
		"storeId":               c.cfg.StoreID,
		"CustomerOrderId":       orderID,
		"merchantTransactionId": merchantTxn,
		"transactionTypeId":     c.cfg.TransactionTypeID,
		"financialEntityId":     0,
		"transitionStatusId":    0,
		"totalAmount":           amount,
		"ipAddress":             first(args["ip_address"], "0.0.0.0"),
		"version":               "1",
		"successUrl":            args["success_url"],
		"failUrl":               args["fail_url"],
		"cancelUrl":             args["cancel_url"],
		"customerName":          name,
		"customerEmail":         email,
		"customerAddress":       address,
		"customerCity":          city,
		"customerState":         state,
		"customerPostcode":      postcode,
		"customerCountry":       country,
		"customerPhone":         phone,
		"productName":           product,
		"valueB":                orderID,
	}
	if valueA := strings.TrimSpace(args["value_a"]); valueA != "" {
		body["valueA"] = valueA
	}
	raw, err := c.postJSON(ctx, c.cfg.endpoints().initialize, body, token, epsHash(merchantTxn, c.cfg.HashKey))
	if err != nil {
		return epsInitResult{}, err
	}
	if msg := first(str(raw, "ErrorMessage"), str(raw, "errorMessage")); msg != "" {
		return epsInitResult{}, fmt.Errorf("EPS initialize failed: %s", msg)
	}
	redirect := first(str(raw, "RedirectURL"), str(raw, "redirectURL"), str(raw, "redirectUrl"))
	if redirect == "" {
		return epsInitResult{}, fmt.Errorf("EPS initialize failed: no redirect")
	}
	return epsInitResult{
		RedirectURL:   redirect,
		TransactionID: first(str(raw, "TransactionId"), str(raw, "transactionId")),
	}, nil
}

func (c *epsClient) Verify(ctx context.Context, merchantTxn string) (epsVerifyResult, error) {
	if err := c.cfg.ready(); err != nil {
		return epsVerifyResult{}, err
	}
	if strings.TrimSpace(merchantTxn) == "" {
		return epsVerifyResult{}, fmt.Errorf("EPS status failed")
	}
	token, err := c.tokenFor(ctx)
	if err != nil {
		return epsVerifyResult{}, err
	}
	q := url.Values{}
	q.Set("merchantTransactionId", merchantTxn)
	endpoint := c.cfg.endpoints().verify + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return epsVerifyResult{}, err
	}
	req.Header.Set("x-hash", epsHash(merchantTxn, c.cfg.HashKey))
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return epsVerifyResult{}, fmt.Errorf("EPS status failed")
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return epsVerifyResult{}, fmt.Errorf("EPS status failed")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return epsVerifyResult{}, fmt.Errorf("EPS status failed")
	}
	if msg := first(str(payload, "ErrorMessage"), str(payload, "errorMessage")); msg != "" {
		return epsVerifyResult{}, fmt.Errorf("EPS status failed")
	}
	amount, _ := parseAmount(first(str(payload, "TotalAmount"), str(payload, "totalAmount")))
	return epsVerifyResult{
		Status:                first(str(payload, "Status"), str(payload, "status")),
		Amount:                amount,
		Currency:              first(str(payload, "Currency"), str(payload, "currency")),
		MerchantTransactionID: first(str(payload, "MerchantTransactionId"), str(payload, "merchantTransactionId"), merchantTxn),
		EPSTransactionID:      first(str(payload, "EPSTransactionId"), str(payload, "EpsTransactionId"), str(payload, "epsTransactionId")),
		FinancialEntity:       first(str(payload, "FinancialEntity"), str(payload, "financialEntity")),
		TransactionDate:       first(str(payload, "TransactionDate"), str(payload, "transactionDate")),
		ErrorMessage:          first(str(payload, "ErrorMessage"), str(payload, "errorMessage")),
	}, nil
}

func (c *epsClient) tokenFor(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.exp) {
		return c.token, nil
	}
	body := map[string]interface{}{"userName": c.cfg.Username, "password": c.cfg.Password}
	raw, err := c.postJSON(ctx, c.cfg.endpoints().token, body, "", epsHash(c.cfg.Username, c.cfg.HashKey))
	if err != nil {
		return "", err
	}
	token := first(str(raw, "token"), str(raw, "Token"))
	if token == "" {
		return "", fmt.Errorf("EPS authentication failed")
	}
	c.token = token
	if exp := first(str(raw, "expireDate"), str(raw, "ExpireDate")); exp != "" {
		if parsed, err := time.Parse(time.RFC3339, exp); err == nil {
			c.exp = parsed.Add(-30 * time.Second)
		}
	}
	if c.exp.IsZero() {
		c.exp = time.Now().Add(10 * time.Minute)
	}
	return c.token, nil
}

func (c *epsClient) postJSON(ctx context.Context, endpoint string, body map[string]interface{}, bearer, hash string) (map[string]interface{}, error) {
	rawBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hash", hash)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("EPS request failed")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("EPS request failed")
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("EPS request failed")
	}
	return payload, nil
}

func parseAmount(raw string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}

func str(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s := scalarString(v); s != "" {
			return s
		}
	}
	for k, v := range m {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			if s := scalarString(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func scalarString(v interface{}) string {
	switch n := v.(type) {
	case string:
		return strings.TrimSpace(n)
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	default:
		return ""
	}
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
