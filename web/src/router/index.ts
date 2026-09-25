import { createRouter, createWebHistory } from "vue-router";
import HomeView from "@/views/HomeView.vue";
import ManageView from "@/views/ManageView.vue";
import CardDetailView from "@/views/CardDetailView.vue";

const router = createRouter({
	history: createWebHistory(),
	routes: [
		{
			path: "/",
			name: "home",
			component: HomeView,
		},
		{
			path: "/manage",
			name: "manage",
			component: ManageView,
		},
		{
			path: "/cards/:id",
			name: "card-detail",
			component: CardDetailView,
		},
	],
});

export default router;
