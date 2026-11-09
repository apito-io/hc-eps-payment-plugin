# hc-eps-payment-plugin

Project-activated EPS hosted checkout.

## Install

```
apito plugin add hc-eps-payment-plugin
```

The engine downloads the signed GitHub Release for this machine. Each project then activates the plugin and stores its own EPS username, password, hash key, merchant id, and store id. Those values are empty in this repository.

The engine install is shared. Each project activates the plugin and stores its own EPS credentials. The Pro host writes `provider=eps` subscriptions only after a server-side status check. Fail and cancel returns do not grant a plan.

Browser return:

`/system/v1/project-payment/projects/{projectId}/providers/eps/callback?result=success&merchant_transaction_id={id}`

Map plans with `provider_products[].provider = "eps"`.
