import { createRouter, createWebHistory, type RouteLocationRaw } from "vue-router";
import { setUnauthorizedHandler } from "@/api";
import { ensureSession, markUnpaired } from "@/composables/useSession";
import { safeNext } from "@/router/params";
import HomeView from "@/views/HomeView.vue";
import CardDetailView from "@/views/CardDetailView.vue";
import CardEditView from "@/views/CardEditView.vue";
import DevicesView from "@/views/DevicesView.vue";
import PairView from "@/views/PairView.vue";

declare module "vue-router" {
	interface RouteMeta {
		/** Reachable without a paired device; the app shell hides navigation. */
		pairing?: boolean;
	}
}

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
		{
			path: "/devices",
			name: "devices",
			component: DevicesView,
		},
		{
			path: "/pair",
			name: "pair",
			component: PairView,
			meta: { pairing: true },
		},
	],
});

// Unpaired browsers go to the pairing screen, which returns them to where
// they were heading; a paired browser (or a server without access control)
// skips it. When the server is unreachable navigation continues and the view
// shows its own error.
router.beforeEach(async (to): Promise<true | RouteLocationRaw> => {
	let session;
	try {
		session = await ensureSession();
	} catch {
		return true;
	}
	if (to.meta.pairing) {
		return session.status === "unpaired" ? true : safeNext(to.query.next);
	}
	if (session.status === "unpaired") return { name: "pair", query: { next: to.fullPath } };
	return true;
});

setUnauthorizedHandler(() => {
	markUnpaired();
	const current = router.currentRoute.value;
	if (!current.meta.pairing) void router.push({ name: "pair", query: { next: current.fullPath } });
});

export default router;
