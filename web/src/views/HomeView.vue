<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import Button from "primevue/button";
import Card from "primevue/card";
import Message from "primevue/message";
import ProgressSpinner from "primevue/progressspinner";
import Tag from "primevue/tag";
import {
	enqueuePrimary,
	enqueueStatus,
	getStatus,
	listCards,
	type ActionCard,
	type StatusSnapshot,
} from "@/api";

type ActionKind = "primary" | "status";
type RequestState = "queued" | "running" | "success" | "error";

const cards = ref<ActionCard[]>([]);
const statuses = ref<Record<string, StatusSnapshot>>({});
const statusErrors = ref<Record<string, boolean>>({});
const requests = ref<Record<string, { action: ActionKind; state: RequestState; message: string }>>(
	{},
);
const loading = ref(true);
const error = ref("");

const defaultFastPollingIntervalSeconds = 10;
const defaultFastPollingWindowSeconds = 120;

const healthyCount = computed(
	() => cards.value.filter((card) => statuses.value[card.id]?.state === "ok").length,
);

function wait(milliseconds: number) {
	return new Promise((resolve) => window.setTimeout(resolve, milliseconds));
}

async function loadDashboard() {
	loading.value = true;
	error.value = "";
	try {
		const loadedCards = await listCards();
		cards.value = loadedCards;
		const loadedStatuses = await Promise.all(
			loadedCards.map(async (card) => {
				try {
					return [card.id, await getStatus(card.id), false] as const;
				} catch {
					return [card.id, undefined, true] as const;
				}
			}),
		);
		const nextStatuses: Record<string, StatusSnapshot> = {};
		const nextStatusErrors: Record<string, boolean> = {};
		for (const [cardID, snapshot, failed] of loadedStatuses) {
			if (snapshot) nextStatuses[cardID] = snapshot;
			if (failed) nextStatusErrors[cardID] = true;
		}
		statuses.value = nextStatuses;
		statusErrors.value = nextStatusErrors;
	} catch (loadError) {
		error.value = loadError instanceof Error ? loadError.message : "Unable to load cards";
	} finally {
		loading.value = false;
	}
}

async function waitForStatusUpdate(card: ActionCard, previousCheckedAt?: string) {
	if (!card.status) return;
	const intervalMilliseconds =
		(card.fastPollingIntervalSeconds || defaultFastPollingIntervalSeconds) * 1000;
	const deadline =
		Date.now() + (card.fastPollingWindowSeconds || defaultFastPollingWindowSeconds) * 1000;

	while (Date.now() < deadline) {
		await wait(intervalMilliseconds);
		try {
			const snapshot = await getStatus(card.id);
			statuses.value[card.id] = snapshot;
			statusErrors.value[card.id] = false;
			if (snapshot.checkedAt !== previousCheckedAt) return true;
		} catch {
			statusErrors.value[card.id] = true;
			return false;
		}
	}
	return false;
}

async function runAction(card: ActionCard, action: ActionKind) {
	if (requests.value[card.id]) return;
	const previousCheckedAt = statuses.value[card.id]?.checkedAt;
	requests.value = {
		...requests.value,
		[card.id]: { action, state: "queued", message: "Queued" },
	};
	try {
		if (action === "primary") await enqueuePrimary(card.id);
		else await enqueueStatus(card.id);
		requests.value[card.id] = { action, state: "running", message: "Running" };
		if (card.status) {
			const updated = await waitForStatusUpdate(card, previousCheckedAt);
			requests.value[card.id] = updated
				? { action, state: "success", message: "Updated" }
				: { action, state: "error", message: "Result not available yet" };
		} else {
			requests.value[card.id] = { action, state: "success", message: "Accepted" };
		}
	} catch (actionError) {
		requests.value[card.id] = {
			action,
			state: "error",
			message: actionError instanceof Error ? actionError.message : "Unable to queue action",
		};
	}
}

function statusView(card: ActionCard) {
	const request = requests.value[card.id];
	if (request?.state === "queued" || request?.state === "running") {
		return { label: request.message, icon: "pi pi-spin pi-spinner", severity: "info" };
	}
	const state = statuses.value[card.id]?.state;
	if (state === "ok") return { label: "Healthy", icon: "pi pi-check-circle", severity: "success" };
	if (state === "fail")
		return { label: "Problem", icon: "pi pi-exclamation-triangle", severity: "danger" };
	return { label: "Unknown", icon: "pi pi-question-circle", severity: "secondary" };
}

function checkedLabel(cardID: string) {
	if (statusErrors.value[cardID]) return "Status data unavailable";
	const checkedAt = statuses.value[cardID]?.checkedAt;
	return checkedAt ? `Last checked ${new Date(checkedAt).toLocaleString()}` : "Not checked yet";
}

function requestLabel(cardID: string) {
	return requests.value[cardID]?.message ?? "";
}

onMounted(loadDashboard);
</script>

<template>
	<main class="overview-page">
		<header class="page-header">
			<div>
				<p class="eyebrow">MARIONETTE CONTROL</p>
				<h1>Overview</h1>
				<p class="page-lede">
					{{ cards.length }} {{ cards.length === 1 ? "action card" : "action cards" }}
					<span v-if="cards.length"> · {{ healthyCount }} healthy</span>
				</p>
			</div>
			<div class="page-actions">
				<Button
					label="Refresh"
					icon="pi pi-refresh"
					severity="secondary"
					outlined
					:loading="loading"
					@click="loadDashboard"
				/>
				<RouterLink class="new-card-link" to="/manage">
					<i class="pi pi-plus" aria-hidden="true"></i>
					<span>New card</span>
				</RouterLink>
			</div>
		</header>

		<Message v-if="error" severity="error" :closable="false">
			<div class="message-content">
				<span>{{ error }}</span>
				<Button label="Try again" text size="small" @click="loadDashboard" />
			</div>
		</Message>

		<div v-if="loading" class="loading-state" role="status" aria-live="polite">
			<ProgressSpinner aria-label="Loading cards" />
			<span>Loading cards...</span>
		</div>

		<section v-else-if="cards.length" class="card-grid" aria-labelledby="cards-heading">
			<h2 id="cards-heading" class="sr-only">Action cards</h2>
			<Card v-for="card in cards" :key="card.id" class="action-card">
				<template #header>
					<div class="card-banner">
						<span class="card-icon" aria-hidden="true">{{ card.icon || "◈" }}</span>
						<Tag :severity="statusView(card).severity">
							<i :class="statusView(card).icon" aria-hidden="true"></i>
							<span>{{ statusView(card).label }}</span>
						</Tag>
					</div>
				</template>
				<template #title>{{ card.name }}</template>
				<template #subtitle>{{ checkedLabel(card.id) }}</template>
				<template #content>
					<p class="card-description">{{ card.description || "No description provided." }}</p>
					<p v-if="requestLabel(card.id)" class="request-feedback" role="status" aria-live="polite">
						{{ requestLabel(card.id) }}
					</p>
				</template>
				<template #footer>
					<div class="card-actions">
						<Button
							label="Run action"
							icon="pi pi-play"
							:loading="
								requests[card.id]?.action === 'primary' && requests[card.id]?.state !== 'error'
							"
							:disabled="Boolean(requests[card.id])"
							@click="runAction(card, 'primary')"
						/>
						<Button
							v-if="card.status"
							label="Check status"
							icon="pi pi-heart"
							severity="secondary"
							outlined
							:loading="
								requests[card.id]?.action === 'status' && requests[card.id]?.state !== 'error'
							"
							:disabled="Boolean(requests[card.id])"
							@click="runAction(card, 'status')"
						/>
					</div>
				</template>
			</Card>
		</section>

		<section v-else class="empty-state" aria-labelledby="empty-title">
			<div class="empty-icon" aria-hidden="true"><i class="pi pi-inbox"></i></div>
			<h2 id="empty-title">No action cards yet</h2>
			<p>Create your first card to start monitoring and controlling a service.</p>
			<RouterLink class="empty-action" to="/manage">
				<i class="pi pi-plus" aria-hidden="true"></i>
				<span>Create your first card</span>
			</RouterLink>
		</section>
	</main>
</template>

<style scoped>
.overview-page {
	max-width: 1200px;
	min-height: calc(100vh - 72px);
	margin: 0 auto;
	padding: clamp(var(--space-6), 5vw, 64px) clamp(var(--space-4), 5vw, 64px);
}

.page-header {
	display: flex;
	align-items: end;
	justify-content: space-between;
	gap: var(--space-8);
	margin-bottom: var(--space-8);
}

.eyebrow {
	margin: 0 0 var(--space-3);
	color: var(--color-accent);
	font-size: 0.75rem;
	font-weight: 800;
	letter-spacing: 0.14em;
}

h1,
h2 {
	margin: 0;
	font-family: var(--font-display);
	font-weight: 500;
}

h1 {
	font-size: clamp(2.5rem, 6vw, 4.5rem);
	line-height: 0.95;
}

.page-lede {
	margin: var(--space-3) 0 0;
	color: var(--color-muted);
	font-size: 1rem;
}

.page-actions,
.card-actions,
.message-content {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: var(--space-3);
}

.page-actions :deep(.p-button),
.card-actions :deep(.p-button),
.new-card-link,
.empty-action {
	font-family: var(--font-ui);
	font-weight: 700;
}

.new-card-link,
.empty-action {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	min-height: 42px;
	padding: 0 var(--space-4);
	border-radius: var(--radius-sm);
	background: var(--color-accent);
	color: #ffffff;
	text-decoration: none;
}

.new-card-link:hover,
.empty-action:hover {
	background: var(--color-accent-strong);
}

.message-content {
	justify-content: space-between;
}

.card-grid {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: var(--space-4);
}

.action-card {
	min-width: 0;
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	box-shadow: var(--shadow-subtle);
}

.action-card :deep(.p-card-body) {
	display: flex;
	flex-direction: column;
	gap: var(--space-4);
	height: 100%;
}

.action-card :deep(.p-card-content) {
	flex: 1;
}

.card-banner {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: var(--space-4) var(--space-4) 0;
}

.card-banner :deep(.p-tag) {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
}

.card-icon {
	display: grid;
	width: 44px;
	height: 44px;
	place-items: center;
	border-radius: var(--radius-sm);
	background: var(--color-accent-soft);
	color: var(--color-accent);
	font-size: 1.35rem;
}

.action-card :deep(.p-card-title) {
	font-family: var(--font-display);
	font-size: 1.4rem;
}

.action-card :deep(.p-card-subtitle) {
	color: var(--color-muted);
	font-family: var(--font-ui);
	font-size: 0.82rem;
}

.card-description {
	min-height: 3rem;
	margin: 0;
	color: var(--color-muted);
	line-height: 1.5;
}

.request-feedback {
	display: inline-flex;
	margin: var(--space-4) 0 0;
	padding: var(--space-2) var(--space-3);
	border-left: 3px solid var(--color-info);
	background: var(--color-info-soft);
	color: var(--color-info);
	font-size: 0.85rem;
	font-weight: 700;
}

.loading-state,
.empty-state {
	display: grid;
	place-items: center;
	gap: var(--space-3);
	min-height: 320px;
	text-align: center;
}

.loading-state {
	color: var(--color-muted);
}

.empty-state {
	padding: var(--space-8);
}

.empty-state p {
	max-width: 34rem;
	margin: 0;
	color: var(--color-muted);
}

.empty-icon {
	display: grid;
	width: 56px;
	height: 56px;
	place-items: center;
	border-radius: 50%;
	background: var(--color-accent-soft);
	color: var(--color-accent);
	font-size: 1.5rem;
}

.sr-only {
	position: absolute;
	width: 1px;
	height: 1px;
	padding: 0;
	overflow: hidden;
	clip: rect(0, 0, 0, 0);
	white-space: nowrap;
	border: 0;
}

@media (max-width: 64rem) {
	.card-grid {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}
}

@media (max-width: 48rem) {
	.overview-page {
		padding: var(--space-6) var(--space-4);
	}

	.page-header {
		align-items: stretch;
		flex-direction: column;
		gap: var(--space-6);
	}

	.page-actions {
		align-items: stretch;
	}

	.page-actions > * {
		flex: 1;
		justify-content: center;
	}

	.card-grid {
		grid-template-columns: 1fr;
	}

	.card-actions > * {
		flex: 1;
	}
}

@media (max-width: 24rem) {
	.page-actions,
	.card-actions {
		flex-direction: column;
		align-items: stretch;
	}

	.page-actions > *,
	.card-actions > * {
		width: 100%;
	}
}
</style>
