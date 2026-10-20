import { createApp } from "vue";
import { createFeedback } from "@/ui/feedback";
import App from "./App.vue";
import router from "./router";

import "./styles/layout.css";
import "./styles/tokens.css";

const app = createApp(App);

app.use(router);
app.use(createFeedback());

app.mount("#app");
