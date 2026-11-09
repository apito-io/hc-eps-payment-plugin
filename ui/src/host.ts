export const PLUGIN_ID = "hc-eps-payment-plugin";

export function sdk() {
  const host = window.ApitoPluginSDK;
  if (!host) {
    throw new Error("ApitoPluginSDK missing");
  }
  return host;
}
