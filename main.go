package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/apito-io/go-apito-plugin-sdk"
)

func main() {
	plugin := sdk.Init("hc-eps-payment-plugin", "0.1.1", "")
	plugin.RegisterMutation("epsInitialize",
		sdk.FieldWithArgs("String", "Start an EPS hosted checkout", map[string]interface{}{
			"merchant_transaction_id": sdk.StringArg("Merchant transaction id"),
			"total_amount":            sdk.FloatArg("Amount"),
			"currency":                sdk.StringArg("Currency"),
			"success_url":             sdk.StringArg("Success return"),
			"fail_url":                sdk.StringArg("Fail return"),
			"cancel_url":              sdk.StringArg("Cancel return"),
		}),
		initializeResolver,
	)
	plugin.RegisterMutation("epsVerify",
		sdk.FieldWithArgs("String", "Verify an EPS merchant transaction", map[string]interface{}{
			"merchant_transaction_id": sdk.StringArg("Merchant transaction id"),
		}),
		verifyResolver,
	)
	plugin.RegisterMutation("epsCancel",
		sdk.FieldWithArgs("String", "EPS does not keep a subscription to cancel", map[string]interface{}{
			"mode": sdk.StringArg("cancel or revoke"),
		}),
		cancelResolver,
	)
	plugin.RegisterMutation("epsReview",
		sdk.FieldWithArgs("String", "Render the public EPS review page", map[string]interface{}{
			"found":        sdk.BooleanArg("Whether the payment row exists"),
			"amount":       sdk.StringArg("Server amount"),
			"currency":     sdk.StringArg("Server currency"),
			"item":         sdk.StringArg("What the payment is for"),
			"error":        sdk.StringArg("Notice to show"),
			"can_proceed":  sdk.BooleanArg("Whether the host allows proceed"),
			"proceed_path": sdk.StringArg("Host proceed path"),
			"status":       sdk.StringArg("Ledger status"),
		}),
		reviewResolver,
	)
	plugin.RegisterRESTAPI(sdk.GETEndpoint("/config", "EPS configuration flags").Build(), configREST)
	plugin.Serve()
}

func initializeResolver(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cfg := configFromArgs(args)
	client := newEPSClient(cfg)
	result, err := client.Initialize(ctx, stringArgs(args))
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(map[string]interface{}{
		"status":                  "pending",
		"redirect_url":            result.RedirectURL,
		"provider_session_id":     result.TransactionID,
		"merchant_transaction_id": stringArgs(args)["merchant_transaction_id"],
		"amount":                  stringArgs(args)["total_amount"],
		"currency":                strings.ToUpper(stringArgs(args)["currency"]),
	})
	return string(raw), nil
}

func verifyResolver(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cfg := configFromArgs(args)
	merchant := stringArgs(args)["merchant_transaction_id"]
	result, err := newEPSClient(cfg).Verify(ctx, merchant)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(map[string]interface{}{
		"status":                  result.Status,
		"amount":                  strconv.FormatFloat(result.Amount, 'f', 2, 64),
		"currency":                result.Currency,
		"client_reference":        result.MerchantTransactionID,
		"merchant_transaction_id": result.MerchantTransactionID,
		"provider_payment_id":     result.EPSTransactionID,
		"financial_entity":        result.FinancialEntity,
		"transaction_date":        result.TransactionDate,
		"error_message":           result.ErrorMessage,
	})
	return string(raw), nil
}

func cancelResolver(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	raw, err := json.Marshal(map[string]interface{}{
		"error": "not supported",
	})
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func reviewResolver(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	raw, err := json.Marshal(map[string]string{"html": renderReviewHTML(args)})
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func configREST(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cfg := configFromArgs(args)
	return map[string]interface{}{
		"enabled":             cfg.Enabled,
		"sandbox":             cfg.Sandbox,
		"username":            cfg.Username,
		"merchant_id":         cfg.MerchantID,
		"store_id":            cfg.StoreID,
		"has_username":        cfg.Username != "",
		"has_password":        cfg.Password != "",
		"has_hash_key":        cfg.HashKey != "",
		"has_merchant_id":     cfg.MerchantID != "",
		"has_store_id":        cfg.StoreID != "",
		"transaction_type_id": cfg.TransactionTypeID,
		"app_return_url":      cfg.AppReturnURL,
	}, nil
}

func configFromArgs(args map[string]interface{}) epsConfig {
	env := envFromArgs(args)
	tx := 1
	if raw := strings.TrimSpace(env["EPS_TRANSACTION_TYPE_ID"]); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			tx = n
		}
	}
	return epsConfig{
		Username:          env["EPS_USERNAME"],
		Password:          env["EPS_PASSWORD"],
		HashKey:           env["EPS_HASH_KEY"],
		MerchantID:        env["EPS_MERCHANT_ID"],
		StoreID:           env["EPS_STORE_ID"],
		Sandbox:           truthy(env["EPS_SANDBOX"]),
		Enabled:           env["EPS_ENABLED"] == "" || truthy(env["EPS_ENABLED"]),
		TransactionTypeID: tx,
		AppReturnURL:      strings.TrimSpace(env["EPS_APP_RETURN_URL"]),
	}
}

func envFromArgs(args map[string]interface{}) map[string]string {
	out := map[string]string{}
	raw, _ := args["context_env_vars"].(map[string]interface{})
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func stringArgs(args map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range args {
		switch n := v.(type) {
		case string:
			out[k] = n
		case float64:
			out[k] = strconv.FormatFloat(n, 'f', -1, 64)
		}
	}
	return out
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
