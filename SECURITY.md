# EPS plugin security

- Credentials live in the project plugin env. The settings tab never reads them back. Blank password and hash key are not sent, so a save does not wipe them.
- `GET /plugin/hc-eps-payment-plugin/config` is protected. It returns booleans and the transaction type id only.
- The browser return is public and untrusted. The Pro host settles `provider=eps` only after `epsVerify` and an amount match. Fail and cancel do not grant a plan.
- Logs and errors do not include the password, hash key, `x-hash`, or EPS token.
