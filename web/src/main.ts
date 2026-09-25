import { createApp } from "vue";
import PrimeVue from "primevue/config";
import Aura from "@primevue/themes/aura";
import App from "./App.vue";
import router from "./router";

import "primeicons/primeicons.css";
import "primeflex/primeflex.css";
import "./styles/tokens.css";

const app = createApp(App);

app.use(router);
app.use(PrimeVue, {
	theme: {
		preset: Aura,
	},
});

app.mount("#app");
