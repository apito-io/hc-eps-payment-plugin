package main

import (
	"strings"
	"testing"
)

func TestRenderReviewHTMLUsesHostFields(t *testing.T) {
	page := renderReviewHTML(map[string]interface{}{
		"found":          true,
		"amount":         "500.00",
		"currency":       "BDT",
		"item":           "paid",
		"item_label":     "Pro",
		"item_detail":    "Subscription for the Rosna software",
		"software_name":  "Rosna",
		"account_name":   "Rosna Restaurant",
		"account_logo":   "https://cdn.example/tenant.png",
		"seller_name":    "Northwind",
		"seller_logo":    "https://cdn.example/company.png",
		"seller_address": "1 Road, Dhaka",
		"seller_phone":   "017",
		"logo_url":       "javascript:alert(1)",
		"reference":      "20261108130740673619964",
		"provider_label": "City Bank",
		"can_proceed":    true,
		"proceed_path":   "/payment/eps/p1/tx/proceed",
		"return_url":     "http://demo.localhost:5056/plans/return",
	})
	for _, want := range []string{
		"500.00",
		"Pro",
		"Rosna Restaurant is paying Northwind for Rosna.",
		"Rosna Restaurant",
		"Subscription for the Rosna software",
		"Billed by",
		"1 Road, Dhaka",
		"017",
		"Payment reference",
		"20261108130740673619964",
		"Continue to City Bank",
		`src="https://cdn.example/company.png"`,
		`src="https://cdn.example/tenant.png"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("missing %q in %s", want, page)
		}
	}
	if !strings.Contains(page, `action="/payment/eps/p1/tx/proceed"`) {
		t.Fatalf("proceed path missing: %s", page)
	}
	if strings.Contains(page, "plans/return") || strings.Contains(page, "localhost:5056") || strings.Contains(page, "javascript:") {
		t.Fatalf("unsafe value leaked into the page: %s", page)
	}
	if strings.Contains(page, "Continue to EPS") {
		t.Fatalf("provider name was hardcoded: %s", page)
	}
}

func TestRenderReviewHTMLUsesProviderIdWhenLabelIsMissing(t *testing.T) {
	page := renderReviewHTML(map[string]interface{}{
		"found":        true,
		"provider":     "eps",
		"can_proceed":  true,
		"proceed_path": "/payment/eps/p1/tx/proceed",
	})
	if !strings.Contains(page, "Continue to EPS") || !strings.Contains(page, "Pay with") {
		t.Fatalf("provider id was not shown: %s", page)
	}
}

func TestRenderReviewHTMLRejectsScriptLogo(t *testing.T) {
	page := renderReviewHTML(map[string]interface{}{
		"found":        true,
		"logo_url":     "javascript:alert(1)",
		"seller_logo":  "javascript:alert(2)",
		"account_logo": "javascript:alert(3)",
	})
	if strings.Contains(page, "javascript:") || strings.Contains(page, "<img") {
		t.Fatalf("logo was not rejected: %s", page)
	}
}

func TestRenderReviewHTMLEscapesError(t *testing.T) {
	page := renderReviewHTML(map[string]interface{}{
		"found":          true,
		"error":          `<script>alert(1)</script>`,
		"seller_address": `<script>alert(2)</script>`,
	})
	if strings.Contains(page, "<script>alert") {
		t.Fatalf("error was not escaped: %s", page)
	}
}
