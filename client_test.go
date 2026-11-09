package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCancelRepliesNotSupported(t *testing.T) {
	raw, err := cancelResolver(context.Background(), map[string]interface{}{"mode": "revoke"})
	if err != nil {
		t.Fatal(err)
	}
	text, _ := raw.(string)
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error"] != "not supported" {
		t.Fatalf("payload=%v", payload)
	}
}

func TestEPSHashMatchesHMACSHA512(t *testing.T) {
	mac := hmac.New(sha512.New, []byte("hash-key"))
	_, _ = mac.Write([]byte("merchant-1"))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got := epsHash("merchant-1", "hash-key"); got != want {
		t.Fatalf("hash=%s", got)
	}
}

func TestEPSInitializeAndVerify(t *testing.T) {
	var sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "hash-key") || strings.Contains(r.Header.Get("x-hash"), "hash-key") && strings.Contains(r.Header.Get("Authorization"), "hash-key") {
			t.Errorf("secret leaked into request log surface")
		}
		switch {
		case strings.Contains(r.URL.Path, "GetToken"):
			if r.Header.Get("x-hash") == "" {
				t.Fatal("missing token hash")
			}
			_, _ = w.Write([]byte(`{"token":"jwt-1","expireDate":"2099-01-01T00:00:00Z"}`))
		case strings.Contains(r.URL.Path, "InitializeEPS"):
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Fatal("missing bearer")
			}
			var sent map[string]interface{}
			if err := json.Unmarshal(body, &sent); err != nil {
				t.Fatal(err)
			}
			if sent["customerName"] != "Ada" || sent["customerEmail"] != "ada@example.com" {
				t.Fatalf("caller customer was dropped: %#v", sent["customerName"])
			}
			if sent["customerPhone"] == "" || sent["customerAddress"] == "" || sent["CustomerOrderId"] != "ORDmerchant-1" {
				t.Fatalf("initialize body missing the fields sandbox success needs: phone=%v address=%v order=%v", sent["customerPhone"], sent["customerAddress"], sent["CustomerOrderId"])
			}
			if _, ok := sent["ProductList"]; ok {
				t.Fatal("ProductList makes sandbox InitializeEPS return 500")
			}
			if sent["ipAddress"] != "0.0.0.0" {
				t.Fatalf("ipAddress=%v", sent["ipAddress"])
			}
			if sent["valueA"] != "tenant-1" || sent["productName"] != "ultra" {
				t.Fatalf("valueA=%v productName=%v", sent["valueA"], sent["productName"])
			}
			_, _ = w.Write([]byte(`{"RedirectURL":"https://pay.example/redirect","TransactionId":"C1"}`))
		case strings.Contains(r.URL.Path, "CheckMerchantTransactionStatus"):
			sawAuth = r.Header.Get("x-hash") != ""
			_, _ = w.Write([]byte(`{"Status":"Success","TotalAmount":"1000.00","MerchantTransactionId":"merchant-1","EPSTransactionId":"2264609071029C","FinancialEntity":"BKash","TransactionDate":"2026-10-29"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := epsConfig{
		Username: "user", Password: "secret", HashKey: "hash-key",
		MerchantID: "m1", StoreID: "s1", Enabled: true, TransactionTypeID: 1, Sandbox: true,
	}
	client := newEPSClient(cfg)
	client.http = srv.Client()
	client.cfg = cfg
	ends := client.cfg.endpoints()
	ends.token = srv.URL + "/v1/Auth/GetToken"
	ends.initialize = srv.URL + "/v1/EPSEngine/InitializeEPS"
	ends.verify = srv.URL + "/v1/EPSEngine/CheckMerchantTransactionStatus"
	client.cfg.Sandbox = true
	// Point the client at the test server by replacing endpoints through a local override.
	orig := client
	_ = orig
	client.http = rewriteDoer{base: srv.URL, next: srv.Client()}

	init, err := client.Initialize(context.Background(), map[string]string{
		"merchant_transaction_id": "merchant-1",
		"total_amount":            "1000",
		"success_url":             "https://api.example/success",
		"fail_url":                "https://api.example/fail",
		"cancel_url":              "https://api.example/cancel",
		"customer_name":           "Ada",
		"customer_email":          "ada@example.com",
		"value_a":                 "tenant-1",
		"product_name":            "ultra",
	})
	if err != nil {
		t.Fatal(err)
	}
	if init.RedirectURL == "" || init.TransactionID != "C1" {
		t.Fatalf("init=%+v", init)
	}
	verified, err := client.Verify(context.Background(), "merchant-1")
	if err != nil {
		t.Fatal(err)
	}
	if !sawAuth || verified.Status != "Success" || verified.Amount != 1000 || verified.EPSTransactionID != "2264609071029C" || verified.FinancialEntity != "BKash" {
		t.Fatalf("verify=%+v auth=%v", verified, sawAuth)
	}
}

func TestEPSVerifyLowercaseAmount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "GetToken") {
			_, _ = w.Write([]byte(`{"token":"jwt-1","expireDate":"2099-01-01T00:00:00Z"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"SUCCESS","totalAmount":"1000.50","merchantTransactionId":"m2"}`))
	}))
	defer srv.Close()
	client := newEPSClient(epsConfig{Username: "u", Password: "p", HashKey: "h", MerchantID: "m", StoreID: "s", Enabled: true, TransactionTypeID: 1})
	client.http = rewriteDoer{base: srv.URL, next: srv.Client()}
	got, err := client.Verify(context.Background(), "m2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "SUCCESS" || got.Amount != 1000.5 || got.MerchantTransactionID != "m2" {
		t.Fatalf("%+v", got)
	}
}

func TestEPSInitializeRejectsInvalidAmountBeforeNetwork(t *testing.T) {
	client := newEPSClient(epsConfig{
		Username: "u", Password: "p", HashKey: "h",
		MerchantID: "m", StoreID: "s", Enabled: true,
	})
	client.http = failDoer{}
	if _, err := client.Initialize(context.Background(), map[string]string{
		"merchant_transaction_id": "m1",
		"total_amount":            "not-a-number",
	}); err == nil {
		t.Fatal("invalid amount must be rejected")
	}
}

func TestEPSVerifyRejectsMissingMerchantTransactionBeforeNetwork(t *testing.T) {
	client := newEPSClient(epsConfig{
		Username: "u", Password: "p", HashKey: "h",
		MerchantID: "m", StoreID: "s", Enabled: true,
	})
	client.http = failDoer{}
	if _, err := client.Verify(context.Background(), " "); err == nil {
		t.Fatal("missing merchant transaction must be rejected")
	}
}

type rewriteDoer struct {
	base string
	next epsDoer
}

type failDoer struct{}

func (failDoer) Do(*http.Request) (*http.Response, error) {
	return nil, io.ErrUnexpectedEOF
}

func (r rewriteDoer) Do(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	u := *req.URL
	switch {
	case strings.Contains(u.Path, "GetToken"):
		u.Path = "/v1/Auth/GetToken"
	case strings.Contains(u.Path, "InitializeEPS"):
		u.Path = "/v1/EPSEngine/InitializeEPS"
	default:
		u.Path = "/v1/EPSEngine/CheckMerchantTransactionStatus"
	}
	u.Scheme = "http"
	u.Host = strings.TrimPrefix(r.base, "http://")
	clone.URL = &u
	clone.Host = u.Host
	return r.next.Do(clone)
}
