<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import Button from "primevue/button";
import Message from "primevue/message";
import ProgressSpinner from "primevue/progressspinner";
import { listCards, type ActionCard as Card, type StatusEvent, type StatusSnapshot } from "@/api";
import ActionCard from "@/components/ActionCard.vue";
import EmptyState from "@/components/EmptyState.vue";
import PageHeader from "@/components/PageHeader.vue";
import { outcomeResult, requestAction } from "@/composables/useActionRequest";
import { supersedes } from "@/composables/useCardStatus";
import { useStatusEvents } from "@/composables/useStatusEvents";
import { messageVisibleMs } from "@/composables/useTransientMessage";
import type { ActionKind, PendingRequest, RequestResult } from "@/types";
import { ACTIONS, EMPTY, FEEDBACK, LOADING } from "@/ui/vocabulary";

const cards = ref<Card[]>([]);
const statuses = ref<Record<string, StatusSnapshot>>({});
// Only queued/running requests live here; a finished request is removed so the
// card's buttons are released, and its outcome is shown briefly via lastResult.
const requests = ref<Record<string, PendingRequest>>({});
const lastResult = ref<Record<string, RequestResult>>({});
const loading = ref(true);
const error = ref("");

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

// Snapshots arrive from three sources (the initial list, the stream and REST
// reads while a request waits or the stream is down); each one is applied
// only when it is not older than what the card already shows.
function applySnapshot(cardID: string, snapshot: StatusSnapshot | undefined) {
	if (!snapshot || !cards.value.some((card) => card.id === cardID)) return;
	if (supersedes(snapshot, statuses.value[cardID])) statuses.value[cardID] = snapshot;
}

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
	} catch (loadError) {
		error.value = loadError instanceof Error ? loadError.message : FEEDBACK.unableToLoadCards;
	} finally {
		loading.value = false;
	}
}

// While the stream is down the list is re-read as a whole: it already carries
// the current status of every card, so one request replaces N status reads.
async function refreshStatuses() {
	try {
		for (const card of await listCards()) applySnapshot(card.id, card.currentStatus);
	} catch {
		// The next tick retries; the cards keep showing their last known status.
	}
}

function applyStatusEvent(event: StatusEvent) {
	applySnapshot(event.cardId, event.snapshot);
}

useStatusEvents().subscribe({
	onStatus: applyStatusEvent,
	onRefresh() {
		if (cards.value.length) void refreshStatuses();
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
		}, messageVisibleMs),
	);
}

function setPending(cardID: string, request: PendingRequest | undefined) {
	const rest = withoutKey(requests.value, cardID);
	requests.value = request ? { ...rest, [cardID]: request } : rest;
}

async function runAction(card: Card, action: ActionKind) {
	if (requests.value[card.id]) return;
	const controller = new AbortController();
	pendingWaits.set(card.id, controller);
	try {
		const outcome = await requestAction(card, action, {
			signal: controller.signal,
			onPhase: (phase) => setPending(card.id, { action, phase }),
			onSnapshot: (snapshot) => applySnapshot(card.id, snapshot),
		});
		const result = outcomeResult(outcome, {
			accepted: FEEDBACK.accepted,
			updated: FEEDBACK.updated,
		});
		if (result) showResult(card.id, result);
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
				<Button v-if="cards.length" v-slot="slotProps" as-child>
					<RouterLink :class="[slotProps.class, 'primary-action-button']" to="/cards/new/edit">
						<i class="pi pi-plus p-button-icon p-button-icon-left" aria-hidden="true" />
						<span class="p-button-label">{{ ACTIONS.newCard }}</span>
					</RouterLink>
				</Button>
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
				<Button v-slot="slotProps" as-child>
					<RouterLink :class="[slotProps.class, 'primary-action-button']" to="/cards/new/edit">
						<i class="pi pi-plus p-button-icon p-button-icon-left" aria-hidden="true" />
						<span class="p-button-label">{{ EMPTY.cards.action }}</span>
					</RouterLink>
				</Button>
			</template>
		</EmptyState>
	</main>
</template>

<style scoped>
.dashboard-grid {
	margin: calc(-1 * var(--space-2));
}
</style>
