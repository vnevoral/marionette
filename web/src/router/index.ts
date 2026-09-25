import { createRouter, createWebHistory } from "vue-router";
import HomeView from "@/views/HomeView.vue";
import CardDetailView from "@/views/CardDetailView.vue";
import CardEditView from "@/views/CardEditView.vue";

const router = createRouter({
	history: createWebHistory(),
	routes: [
		{
			path: "/",
			name: "home",
			component: HomeView,
		},
		{
			path: "/cards/new/edit",
			name: "card-new",
			component: CardEditView,
		},
		{
			path: "/cards/:id",
			name: "card-detail",
			component: CardDetailView,
		},
		{
			path: "/cards/:id/edit",
			name: "card-edit",
			component: CardEditView,
		},
	],
});

export default router;
