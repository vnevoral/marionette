import { createApp } from "vue";
import PrimeVue from "primevue/config";
import ConfirmationService from "primevue/confirmationservice";
import MarionettePreset from "@/theme/preset";
import App from "./App.vue";
import router from "./router";

import "primeicons/primeicons.css";
import "primeflex/primeflex.css";
import "./styles/tokens.css";

const app = createApp(App);

app.use(router);
app.use(ConfirmationService);
app.use(PrimeVue, {
	theme: {
		preset: MarionettePreset,
		// The UI ships one light scheme; do not follow the OS dark mode.
		options: { darkModeSelector: false },
	},
});

app.mount("#app");
