package main

import (
	"html/template"
	"net/url"
	"strings"
)

var reviewPageTmpl = template.Must(template.New("review").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{if .Title}}Pay {{.Title}}{{else}}Review payment{{end}}</title>
<style>
html,body{height:100%}
body{margin:0;display:flex;align-items:center;justify-content:center;background:#eef2f6;color:#14202b;font-family:system-ui,-apple-system,sans-serif}
main{width:min(28rem,calc(100% - 2rem));background:#fff;border-radius:20px;padding:1.75rem 1.5rem 1.5rem;box-shadow:0 16px 40px rgba(20,32,43,.08)}
header{display:flex;gap:.9rem;align-items:center}
img{width:48px;height:48px;border-radius:12px;object-fit:cover;background:#f3f5f7}
.brand{margin:0;font-size:.78rem;letter-spacing:.04em;text-transform:uppercase;color:#5c6b78;font-weight:650}
h1{margin:.15rem 0 0;font-size:1.35rem}
.lead{margin:.9rem 0 0;color:#52616f;line-height:1.45}
.payer{margin:1rem 0 0;display:flex;gap:.75rem;align-items:center;padding:.8rem;border-radius:14px;background:#f6f8fa}
.payer img{width:40px;height:40px;border-radius:10px}
.payer strong{display:block;font-size:1.02rem}
.amount{margin:1.15rem 0 0;padding:1rem 0;border-top:1px solid #e6ebf0;border-bottom:1px solid #e6ebf0;display:flex;align-items:baseline;gap:.4rem}
.amount b{font-size:1.85rem;letter-spacing:-.03em}
.amount span{color:#52616f;font-weight:650}
dl{margin:1rem 0 0;display:grid;grid-template-columns:9.2rem 1fr;gap:.55rem .75rem}
dt{color:#52616f}
dd{margin:0;font-weight:650}
.ref{font-family:ui-monospace,monospace;font-weight:600;font-size:.92rem;word-break:break-all}
.err{margin-top:1rem;color:#c62828}
button{width:100%;margin-top:1.35rem;padding:.95rem 1rem;border:0;border-radius:12px;background:#0f8a45;color:#fff;font:inherit;font-weight:650;cursor:pointer}
.fine{margin:.75rem 0 0;color:#6b7a88;font-size:.82rem;line-height:1.45}
</style>
</head>
<body>
<main>
{{if .Found}}
<header>
{{if .HeaderLogo}}<img src="{{.HeaderLogo}}" alt="">{{end}}
<div>
{{if .Seller}}<p class="brand">{{.Seller}}</p>{{else if .Software}}<p class="brand">{{.Software}}</p>{{end}}
<h1>Review payment</h1>
</div>
</header>
<p class="lead">{{.Lead}}</p>
{{if .Account}}
<div class="payer">
{{if .AccountLogo}}<img src="{{.AccountLogo}}" alt="">{{end}}
<div>
<p class="brand">Account</p>
<strong>{{.Account}}</strong>
</div>
</div>
{{end}}
<p class="amount"><b>{{.Amount}}</b> <span>{{.Currency}}</span></p>
<dl>
{{if .Software}}<dt>Software</dt><dd>{{.Software}}</dd>{{end}}
<dt>Plan</dt><dd>{{.Plan}}</dd>
{{if .Detail}}<dt>For</dt><dd>{{.Detail}}</dd>{{end}}
{{if .Seller}}<dt>Billed by</dt><dd>{{.Seller}}</dd>{{end}}
{{if .Reference}}<dt>Payment reference</dt><dd class="ref">{{.Reference}}</dd>{{end}}
{{if .Provider}}<dt>Pay with</dt><dd>{{.Provider}}</dd>{{end}}
</dl>
{{if .Error}}<p class="err">{{.Error}}</p>{{end}}
{{if .CanProceed}}
<form method="post" action="{{.ProceedPath}}">
<button type="submit">{{.Button}}</button>
</form>
{{end}}
{{if .SellerLines}}<p class="fine">{{.SellerLines}}</p>{{end}}
<p class="fine">{{.Fine}}</p>
{{else}}
<h1>Review payment</h1>
<p class="err">This payment could not be found.</p>
{{end}}
</main>
</body>
</html>
`))

type reviewView struct {
	Found       bool
	Title       string
	Seller      string
	HeaderLogo  template.URL
	Software    string
	Lead        string
	Account     string
	AccountLogo template.URL
	Amount      string
	Currency    string
	Plan        string
	Detail      string
	SellerLines string
	Reference   string
	Provider    string
	Button      string
	Fine        string
	Error       string
	CanProceed  bool
	ProceedPath template.URL
}

func renderReviewHTML(args map[string]interface{}) string {
	software := stringArg(args, "software_name")
	seller := stringArg(args, "seller_name")
	account := stringArg(args, "account_name")
	provider := stringArg(args, "provider_label")
	if provider == "" {
		provider = displayToken(stringArg(args, "provider"))
	}
	plan := stringArg(args, "item_label")
	if plan == "" {
		plan = stringArg(args, "item")
	}
	if plan == "" {
		plan = "Payment"
	}
	header := httpsLogo(stringArg(args, "seller_logo"))
	if header == "" {
		header = httpsLogo(stringArg(args, "logo_url"))
	}
	title := seller
	if title == "" {
		title = software
	}
	button := "Continue"
	fine := "Nothing is charged on this page."
	if provider != "" {
		button = "Continue to " + provider
		fine = provider + " asks you to confirm this amount. Nothing is charged on this page."
	}
	if stringArg(args, "reference") != "" {
		fine = "The payment reference identifies this checkout. " + fine
	}
	view := reviewView{
		Found:       boolArg(args, "found"),
		Title:       title,
		Seller:      seller,
		HeaderLogo:  template.URL(header),
		Software:    software,
		Lead:        reviewLead(account, seller, software),
		Account:     account,
		AccountLogo: template.URL(httpsLogo(stringArg(args, "account_logo"))),
		Amount:      stringArg(args, "amount"),
		Currency:    stringArg(args, "currency"),
		Plan:        plan,
		Detail:      stringArg(args, "item_detail"),
		SellerLines: sellerContact(stringArg(args, "seller_address"), stringArg(args, "seller_phone"), stringArg(args, "seller_extra")),
		Reference:   stringArg(args, "reference"),
		Provider:    provider,
		Button:      button,
		Fine:        fine,
		Error:       stringArg(args, "error"),
		CanProceed:  boolArg(args, "can_proceed") && stringArg(args, "proceed_path") != "",
		ProceedPath: template.URL(stringArg(args, "proceed_path")),
	}
	var buf strings.Builder
	_ = reviewPageTmpl.Execute(&buf, view)
	return buf.String()
}

func reviewLead(account, seller, software string) string {
	switch {
	case account != "" && seller != "" && software != "" && !strings.EqualFold(seller, software):
		return account + " is paying " + seller + " for " + software + "."
	case account != "" && software != "":
		return account + " is paying for " + software + "."
	case software != "":
		return "You are paying for " + software + "."
	default:
		return "Check the amount before you continue."
	}
}

func sellerContact(address, phone, extra string) string {
	var parts []string
	for _, part := range []string{address, phone, extra} {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " · ")
}

func displayToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.FieldsFunc(strings.ReplaceAll(raw, "-", "_"), func(r rune) bool {
		return r == '_' || r == ' '
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if len([]rune(part)) <= 3 {
			out = append(out, strings.ToUpper(part))
			continue
		}
		lower := strings.ToLower(part)
		out = append(out, strings.ToUpper(lower[:1])+lower[1:])
	}
	return strings.Join(out, " ")
}

func httpsLogo(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil || parsed.Host == "" || !strings.EqualFold(parsed.Scheme, "https") {
		return ""
	}
	return raw
}

func stringArg(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	switch v := args[key].(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func boolArg(args map[string]interface{}, key string) bool {
	if args == nil {
		return false
	}
	switch v := args[key].(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes":
			return true
		}
	}
	return false
}
