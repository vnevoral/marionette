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
let statusRefreshGeneration = 0;

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
		const nextStatuses: Record<string, StatusSnapshot> = {};
		for (const card of loadedCards) {
			if (card.currentStatus) nextStatuses[card.id] = card.currentStatus;
		}
		statuses.value = nextStatuses;
		statusErrors.value = {};
		loading.value = false;
		void refreshStatuses(loadedCards);
	} catch (loadError) {
		error.value = loadError instanceof Error ? loadError.message : "Unable to load cards";
		loading.value = false;
	} finally {
		loading.value = false;
	}
}

async function refreshStatuses(cardsToRefresh: ActionCard[]) {
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
	if (!card.status) {
		return {
			label: "No status check",
			icon: "pi pi-minus-circle",
			severity: "secondary",
			tone: "unknown",
		};
	}
	const request = requests.value[card.id];
	if (request?.state === "queued" || request?.state === "running") {
		return {
			label: request.message,
			icon: "pi pi-spin pi-spinner",
			severity: "info",
			tone: "info",
		};
	}
	const state = statuses.value[card.id]?.state;
	if (state === "ok")
		return { label: "Healthy", icon: "pi pi-check-circle", severity: "success", tone: "healthy" };
	if (state === "fail")
		return {
			label: "Problem",
			icon: "pi pi-exclamation-triangle",
			severity: "danger",
			tone: "problem",
		};
	return {
		label: "Unknown",
		icon: "pi pi-question-circle",
		severity: "secondary",
		tone: "unknown",
	};
}

function requestLabel(cardID: string) {
	return requests.value[cardID]?.message ?? "";
}

onMounted(loadDashboard);
</script>

<template>
	<main class="overview-page">
		<header
			class="page-header flex flex-column md:flex-row align-items-start md:align-items-end justify-content-between gap-6 mb-8"
		>
			<div>
				<p class="eyebrow">MARIONETTE CONTROL</p>
				<h1>Overview</h1>
				<p class="page-lede">
					{{ cards.length }} {{ cards.length === 1 ? "action card" : "action cards" }}
					<span v-if="cards.length"> · {{ healthyCount }} healthy</span>
				</p>
			</div>
			<div class="page-actions flex flex-wrap align-items-center gap-3">
				<Button
					label="Refresh"
					icon="pi pi-refresh"
					severity="secondary"
					outlined
					:loading="loading"
					@click="loadDashboard"
				/>
				<RouterLink v-if="cards.length" class="new-card-link" to="/manage">
					<i class="pi pi-plus" aria-hidden="true"></i>
					<span>New card</span>
				</RouterLink>
			</div>
		</header>

		<Message v-if="error" severity="error" :closable="false">
			<div class="message-content flex align-items-center flex-wrap justify-content-between gap-3">
				<span>{{ error }}</span>
				<Button label="Try again" text size="small" @click="loadDashboard" />
			</div>
		</Message>

		<div v-if="loading" class="loading-state" role="status" aria-live="polite">
			<ProgressSpinner aria-label="Loading cards" />
			<span>Loading cards...</span>
		</div>

		<section v-else-if="cards.length" class="dashboard-grid grid" aria-labelledby="cards-heading">
			<h2 id="cards-heading" class="sr-only">Action cards</h2>
			<div v-for="card in cards" :key="card.id" class="col-12 md:col-6 lg:col-4 p-2">
				<Card class="action-card h-full">
					<template #header>
						<div class="card-banner">
							<div class="card-icon" aria-hidden="true">
								<i :class="card.icon || 'pi pi-desktop'" />
							</div>
							<Tag
								:severity="statusView(card).severity"
								:class="`status-tag status-${statusView(card).tone}`"
							>
								<i :class="statusView(card).icon" aria-hidden="true"></i>
								<span>{{ statusView(card).label }}</span>
							</Tag>
						</div>
					</template>
					<template #title>
						<div class="card-title-row">
							<span class="card-title">{{ card.name }}</span>
						</div>
					</template>
					<template #content>
						<p class="card-description">{{ card.description || "No description provided." }}</p>
						<p
							v-if="requestLabel(card.id)"
							class="request-feedback"
							role="status"
							aria-live="polite"
						>
							{{ requestLabel(card.id) }}
						</p>
					</template>
					<template #footer>
						<div class="card-footer">
							<RouterLink class="details-link" :to="`/cards/${encodeURIComponent(card.id)}`">
								<span>View details</span>
								<i class="pi pi-arrow-up-right" aria-hidden="true"></i>
							</RouterLink>
							<div class="card-action-buttons">
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
						</div>
					</template>
				</Card>
			</div>
		</section>

		<section
			v-else
			class="empty-state flex flex-column align-items-center justify-content-center gap-3 p-8 text-center"
			aria-labelledby="empty-title"
		>
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
	margin: 0 auto;
	padding: clamp(var(--space-6), 5vw, 64px) clamp(var(--space-4), 5vw, 64px);
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

.action-card {
	min-width: 0;
	height: 100%;
	border: 1px solid var(--color-border);
	background: var(--color-surface);
	box-shadow: var(--shadow-subtle);
}

.dashboard-grid {
	margin: -var(--space-2);
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

.card-title-row {
	display: flex;
	min-width: 0;
	flex-direction: column;
	gap: var(--space-1);
}

.card-title {
	overflow: hidden;
	color: var(--color-ink);
	text-overflow: ellipsis;
	white-space: nowrap;
}

.action-card :deep(.p-card-title) {
	margin: 0;
	font-family: var(--font-display);
	font-size: 1.45rem;
	font-weight: 500;
}

.card-description {
	margin: 0;
	color: var(--color-muted);
	line-height: 1.5;
}

.card-footer {
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
}

.details-link {
	display: inline-flex;
	align-items: center;
	align-self: flex-start;
	gap: var(--space-2);
	color: var(--color-accent-strong);
	font-size: 0.88rem;
	font-weight: 700;
	text-decoration: none;
}

.details-link:hover {
	color: var(--color-ink);
	text-decoration: underline;
}

.card-action-buttons {
	display: flex;
	flex-wrap: wrap;
	gap: var(--space-2);
}

.card-action-buttons :deep(.p-button) {
	flex: 1 1 auto;
}

.card-banner :deep(.status-tag) {
	border: 1px solid transparent;
	font-weight: 800;
}

.card-banner :deep(.status-healthy) {
	border-color: #1f6b50;
	background: #1f6b50;
	color: #ffffff;
}

.card-banner :deep(.status-problem) {
	border-color: #9f3f3a;
	background: #9f3f3a;
	color: #ffffff;
}

.card-banner :deep(.status-unknown) {
	border-color: var(--color-border-strong);
	background: var(--color-surface-raised);
	color: var(--color-ink);
}

.card-banner :deep(.status-info) {
	border-color: var(--color-info);
	background: var(--color-info);
	color: #ffffff;
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

.loading-state {
	min-height: 320px;
	color: var(--color-muted);
}

.empty-state {
	min-height: 320px;
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
}

@media (max-width: 48rem) {
	.overview-page {
		padding: var(--space-6) var(--space-4);
	}

	.page-actions > * {
		flex: 1;
		justify-content: center;
	}
}

@media (max-width: 24rem) {
	.page-actions,
	.card-action-buttons {
		flex-direction: column;
		align-items: stretch;
	}

	.page-actions > *,
	.card-action-buttons > * {
		width: 100%;
	}
}
</style>
