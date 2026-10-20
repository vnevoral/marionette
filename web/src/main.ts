import { createApp } from "vue";
import PrimeVue from "primevue/config";
import MarionettePreset from "@/theme/preset";
import { createFeedback } from "@/ui/feedback";
import App from "./App.vue";
import router from "./router";

import "primeicons/primeicons.css";
import "./styles/layout.css";
import "./styles/tokens.css";

const app = createApp(App);

app.use(router);
app.use(createFeedback());
app.use(PrimeVue, {
	theme: {
		preset: MarionettePreset,
		// The UI ships one light scheme; do not follow the OS dark mode.
		options: { darkModeSelector: false },
	},
});

app.mount("#app");
