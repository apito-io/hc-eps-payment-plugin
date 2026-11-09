import { PLUGIN_ID, sdk } from "./host";
import EPSProviderTab from "./provider";

sdk().register({
  name: PLUGIN_ID,
  version: "0.1.0",
  displayName: "EPS",
  routes: [],
  components: {
    slots: {
      "payment-providers": EPSProviderTab,
    },
  },
});
