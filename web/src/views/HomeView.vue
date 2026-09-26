<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import Button from "primevue/button";
import Message from "primevue/message";
import ProgressSpinner from "primevue/progressspinner";
import {
	enqueuePrimary,
	enqueueStatus,
	getStatus,
	listCards,
	type ActionCard as Card,
	type StatusEvent,
	type StatusSnapshot,
} from "@/api";
import ActionCard from "@/components/ActionCard.vue";
import EmptyState from "@/components/EmptyState.vue";
import PageHeader from "@/components/PageHeader.vue";
import { waitForNewerStatus } from "@/composables/useCardStatus";
import { useStatusEvents } from "@/composables/useStatusEvents";
import type { ActionKind, PendingRequest, RequestResult } from "@/types";
import { ACTIONS, EMPTY, FEEDBACK, LOADING } from "@/ui/vocabulary";

const cards = ref<Card[]>([]);
const statuses = ref<Record<string, StatusSnapshot>>({});
const statusErrors = ref<Record<string, boolean>>({});
// Only queued/running requests live here; a finished request is removed so the
// card's buttons are released, and its outcome is shown briefly via lastResult.
const requests = ref<Record<string, PendingRequest>>({});
const lastResult = ref<Record<string, RequestResult>>({});
const loading = ref(true);
const error = ref("");

const defaultFastPollingWindowSeconds = 120;
const resultVisibleMs = 4000;
let statusRefreshGeneration = 0;
const resultTimers = new Map<string, ReturnType<typeof setTimeout>>();
const pendingWaits = new Map<string, AbortController>();

const healthyCount = computed(
	() => cards.value.filter((card) => statuses.value[card.id]?.state === "ok").length,
);
const lede = computed(() => {
	const count = cards.value.length;
	const noun = count === 1 ? "action card" : "action cards";
	return count ? `${count} ${noun} · ${healthyCount.value} healthy` : `${count} ${noun}`;
});

async function loadDashboard() {
	loading.value = true;
	error.value = "";
	try {
		const loadedCards = await listCards();
		cards.value = loadedCards;
		const nextStatuses: Record<string, StatusSnapshot> = {};
		for (const card of loadedCards) {
			if (card.currentStatus) nextStatuses[card.id] = card.currentStatus;
		}
		statuses.value = nextStatuses;
		statusErrors.value = {};
		loading.value = false;
		void refreshStatuses(loadedCards);
	} catch (loadError) {
		error.value = loadError instanceof Error ? loadError.message : FEEDBACK.unableToLoadCards;
	} finally {
		loading.value = false;
	}
}

async function refreshStatuses(cardsToRefresh: Card[]) {
	const generation = ++statusRefreshGeneration;
	const statusResults = await Promise.all(
		cardsToRefresh
			.filter((card) => card.status)
			.map(async (card) => {
				try {
					return [card.id, await getStatus(card.id), false] as const;
				} catch {
					return [card.id, undefined, true] as const;
				}
			}),
	);
	if (generation !== statusRefreshGeneration) return;
	for (const [cardID, snapshot, failed] of statusResults) {
		if (snapshot) statuses.value[cardID] = snapshot;
		statusErrors.value[cardID] = failed;
	}
}

function applyStatusEvent(event: StatusEvent) {
	if (!cards.value.some((card) => card.id === event.cardId)) return;
	statusRefreshGeneration++;
	statuses.value[event.cardId] = event.snapshot;
	statusErrors.value[event.cardId] = false;
}

useStatusEvents().subscribe({
	onStatus: applyStatusEvent,
	onRefresh() {
		if (cards.value.length) void refreshStatuses(cards.value);
	},
});

function withoutKey<T>(record: Record<string, T>, key: string): Record<string, T> {
	const copy = { ...record };
	delete copy[key];
	return copy;
}

function showResult(cardID: string, result: RequestResult) {
	lastResult.value = { ...lastResult.value, [cardID]: result };
	clearTimeout(resultTimers.get(cardID));
	resultTimers.set(
		cardID,
		setTimeout(() => {
			resultTimers.delete(cardID);
			lastResult.value = withoutKey(lastResult.value, cardID);
		}, resultVisibleMs),
	);
}

function setPending(cardID: string, request: PendingRequest | undefined) {
	const rest = withoutKey(requests.value, cardID);
	requests.value = request ? { ...rest, [cardID]: request } : rest;
}

async function runAction(card: Card, action: ActionKind) {
	if (requests.value[card.id]) return;
	const previousCheckedAt = statuses.value[card.id]?.checkedAt;
	setPending(card.id, { action, phase: "queued" });
	const controller = new AbortController();
	pendingWaits.set(card.id, controller);
	try {
		if (action === "primary") await enqueuePrimary(card.id);
		else await enqueueStatus(card.id);
		if (!card.status) {
			showResult(card.id, { tone: "success", message: FEEDBACK.accepted });
			return;
		}
		setPending(card.id, { action, phase: "running" });
		const result = await waitForNewerStatus(card.id, previousCheckedAt, {
			signal: controller.signal,
			maxWaitMs: (card.fastPollingWindowSeconds || defaultFastPollingWindowSeconds) * 1000,
			onSnapshot(snapshot) {
				statusRefreshGeneration++;
				statuses.value[card.id] = snapshot;
				statusErrors.value[card.id] = false;
			},
			onError() {
				statusErrors.value[card.id] = true;
			},
		});
		if (result === "aborted") return;
		showResult(
			card.id,
			result === "updated"
				? { tone: "success", message: FEEDBACK.updated }
				: { tone: "error", message: FEEDBACK.resultUnavailable },
		);
	} catch (actionError) {
		showResult(card.id, {
			tone: "error",
			message: actionError instanceof Error ? actionError.message : FEEDBACK.unableToQueue,
		});
	} finally {
		pendingWaits.delete(card.id);
		setPending(card.id, undefined);
	}
}

onMounted(() => {
	void loadDashboard();
});

onBeforeUnmount(() => {
	for (const controller of pendingWaits.values()) controller.abort();
	pendingWaits.clear();
	for (const timer of resultTimers.values()) clearTimeout(timer);
	resultTimers.clear();
});
</script>

<template>
	<main class="page">
		<PageHeader eyebrow="Marionette control" title="Overview" :lede="lede">
			<template #actions>
				<Button
					:label="ACTIONS.refresh"
					icon="pi pi-refresh"
					severity="secondary"
					outlined
					:loading="loading"
					@click="loadDashboard"
				/>
				<RouterLink v-if="cards.length" class="primary-action-link" to="/cards/new/edit">
					<i class="pi pi-plus" aria-hidden="true" />
					<span>{{ ACTIONS.newCard }}</span>
				</RouterLink>
			</template>
		</PageHeader>

		<Message v-if="error" severity="error" :closable="false">
			<div class="flex flex-wrap align-items-center justify-content-between gap-3">
				<span>{{ error }}</span>
				<Button :label="ACTIONS.tryAgain" text size="small" @click="loadDashboard" />
			</div>
		</Message>

		<div v-if="loading" class="loading-state" role="status" aria-live="polite">
			<ProgressSpinner :aria-label="LOADING.cards" />
			<span>{{ LOADING.cards }}</span>
		</div>

		<section v-else-if="cards.length" class="dashboard-grid grid" aria-labelledby="cards-heading">
			<h2 id="cards-heading" class="sr-only">Action cards</h2>
			<div v-for="card in cards" :key="card.id" class="col-12 md:col-6 lg:col-4 p-2">
				<ActionCard
					:card="card"
					:status="statuses[card.id]"
					:pending="requests[card.id]"
					:result="lastResult[card.id]"
					@run="runAction(card, 'primary')"
					@check="runAction(card, 'status')"
				/>
			</div>
		</section>

		<EmptyState v-else icon="pi pi-inbox" :title="EMPTY.cards.title" :body="EMPTY.cards.body">
			<template #action>
				<RouterLink class="primary-action-link" to="/cards/new/edit">
					<i class="pi pi-plus" aria-hidden="true" />
					<span>{{ EMPTY.cards.action }}</span>
				</RouterLink>
			</template>
		</EmptyState>
	</main>
</template>

<style scoped>
.dashboard-grid {
	margin: calc(-1 * var(--space-2));
}
</style>
