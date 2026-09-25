<script setup lang="ts">
import { onMounted, ref } from "vue";
import Button from "primevue/button";
import Card from "primevue/card";
import Message from "primevue/message";
import ProgressSpinner from "primevue/progressspinner";
import Tag from "primevue/tag";
import {
	getStatus,
	listCards,
	enqueuePrimary,
	enqueueStatus,
	type ActionCard,
	type StatusSnapshot,
} from "@/api";

const cards = ref<ActionCard[]>([]);
const statuses = ref<Record<string, StatusSnapshot>>({});
const pending = ref<Record<string, "primary" | "status"> | undefined>({});
const loading = ref(true);
const error = ref("");

const defaultFastPollingIntervalSeconds = 10;
const defaultFastPollingWindowSeconds = 120;

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
					return [card.id, await getStatus(card.id)] as const;
				} catch {
					return [card.id, undefined] as const;
				}
			}),
		);
		statuses.value = Object.fromEntries(
			loadedStatuses.filter((entry): entry is [string, StatusSnapshot] => entry[1] !== undefined),
		);
	} catch (loadError) {
		error.value = loadError instanceof Error ? loadError.message : "Unable to load dashboard";
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
			if (snapshot.checkedAt !== previousCheckedAt) return;
		} catch {
			return;
		}
	}
}

async function runAction(card: ActionCard, action: "primary" | "status") {
	if (pending.value?.[card.id]) return;
	const previousCheckedAt = statuses.value[card.id]?.checkedAt;
	pending.value = { ...pending.value, [card.id]: action };
	error.value = "";
	try {
		if (action === "primary") await enqueuePrimary(card.id);
		else await enqueueStatus(card.id);
		if (card.status) await waitForStatusUpdate(card, previousCheckedAt);
	} catch (actionError) {
		error.value = actionError instanceof Error ? actionError.message : "Unable to queue action";
	} finally {
		const nextPending = { ...pending.value };
		delete nextPending[card.id];
		pending.value = nextPending;
	}
}

onMounted(loadDashboard);

function stateLabel(card: ActionCard) {
	return pending.value?.[card.id] ? "Running" : (statuses.value[card.id]?.state ?? "unknown");
}

function stateSeverity(state: string) {
	if (state === "ok") return "success";
	if (state === "fail") return "danger";
	return "secondary";
}

function checkedLabel(cardID: string) {
	const checkedAt = statuses.value[cardID]?.checkedAt;
	return checkedAt ? `Checked ${new Date(checkedAt).toLocaleString()}` : "No status check yet";
}
</script>

<template>
	<main class="dashboard-shell">
		<header class="dashboard-header">
			<div>
				<p class="eyebrow">MARIONETTE CONTROL</p>
				<h1>Action cards</h1>
				<p class="lede">A quiet view of the machines and services you can move.</p>
			</div>
			<Button
				label="Refresh"
				icon="pi pi-refresh"
				severity="secondary"
				outlined
				:loading="loading"
				@click="loadDashboard"
			/>
		</header>

		<Message v-if="error" severity="error" :closable="false">{{ error }}</Message>
		<div v-if="loading" class="loading-state">
			<ProgressSpinner aria-label="Loading dashboard" />
		</div>
		<section v-else-if="cards.length" class="card-grid" aria-label="Action cards">
			<Card v-for="card in cards" :key="card.id" class="action-card">
				<template #header>
					<div class="card-banner">
						<span class="card-icon" aria-hidden="true">{{ card.icon || "◈" }}</span>
						<Tag :value="stateLabel(card)" :severity="stateSeverity(stateLabel(card))" />
					</div>
				</template>
				<template #title>{{ card.name }}</template>
				<template #subtitle>{{ checkedLabel(card.id) }}</template>
				<template #content
					><p class="card-description">
						{{ card.description || "No description provided." }}
					</p></template
				>
				<template #footer>
					<div class="card-actions">
						<Button
							label="Run action"
							icon="pi pi-play"
							:loading="pending?.[card.id] === 'primary'"
							:disabled="Boolean(pending?.[card.id])"
							@click="runAction(card, 'primary')"
						/>
						<Button
							v-if="card.status"
							label="Check status"
							icon="pi pi-heart"
							severity="secondary"
							outlined
							:loading="pending?.[card.id] === 'status'"
							:disabled="Boolean(pending?.[card.id])"
							@click="runAction(card, 'status')"
						/>
					</div>
				</template>
			</Card>
		</section>
		<div v-else class="empty-state">
			<i class="pi pi-inbox" aria-hidden="true"></i>
			<h2>No action cards yet</h2>
			<p>Create a card to see it here.</p>
		</div>
	</main>
</template>

<style scoped>
:global(body) {
	margin: 0;
	background: #f4f1eb;
	color: #1e2926;
	font-family: Georgia, "Times New Roman", serif;
}
.dashboard-shell {
	min-height: 100vh;
	padding: clamp(2rem, 6vw, 5.5rem) clamp(1rem, 5vw, 5rem);
	background: radial-gradient(circle at top right, #d5e4d9 0, transparent 35rem), #f4f1eb;
}
.dashboard-header {
	display: flex;
	align-items: flex-end;
	justify-content: space-between;
	gap: 2rem;
	max-width: 78rem;
	margin: 0 auto 3rem;
}
.eyebrow {
	margin: 0 0 0.75rem;
	color: #a24b32;
	font-family: "Trebuchet MS", sans-serif;
	font-size: 0.75rem;
	font-weight: 700;
	letter-spacing: 0.14em;
}
h1,
h2 {
	margin: 0;
	font-weight: 500;
}
h1 {
	font-size: clamp(2.5rem, 6vw, 5rem);
	line-height: 0.95;
}
.lede {
	max-width: 32rem;
	margin: 1rem 0 0;
	color: #53615b;
	font-size: 1.1rem;
}
.dashboard-header :deep(.p-button),
.action-card :deep(.p-card-title),
.action-card :deep(.p-card-subtitle),
.card-description,
.card-actions :deep(.p-button) {
	font-family: "Trebuchet MS", sans-serif;
}
.card-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(min(100%, 19rem), 1fr));
	gap: 1.25rem;
	max-width: 78rem;
	margin: 0 auto;
}
.action-card {
	border: 1px solid #d7d3ca;
	box-shadow: 0 1rem 2.5rem rgb(44 58 50 / 8%);
}
.action-card :deep(.p-card-body) {
	gap: 1rem;
	height: 100%;
}
.card-banner {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 1.25rem 1.25rem 0;
}
.card-icon {
	display: grid;
	width: 3rem;
	height: 3rem;
	place-items: center;
	border-radius: 50%;
	background: #e1eadf;
	color: #a24b32;
	font-size: 1.5rem;
}
.action-card :deep(.p-card-title) {
	font-size: 1.45rem;
}
.action-card :deep(.p-card-subtitle) {
	font-size: 0.78rem;
}
.card-description {
	min-height: 3rem;
	margin: 0;
	color: #53615b;
	line-height: 1.5;
}
.card-actions {
	display: flex;
	flex-wrap: wrap;
	gap: 0.65rem;
}
.loading-state,
.empty-state {
	display: grid;
	place-items: center;
	max-width: 78rem;
	min-height: 18rem;
	margin: 0 auto;
	text-align: center;
}
.empty-state i {
	margin-bottom: 1rem;
	color: #a24b32;
	font-size: 2rem;
}
.empty-state p {
	margin-top: 0.5rem;
	color: #53615b;
	font-family: "Trebuchet MS", sans-serif;
}
@media (max-width: 36rem) {
	.dashboard-header {
		align-items: stretch;
		flex-direction: column;
		margin-bottom: 2rem;
	}
	.dashboard-header :deep(.p-button) {
		align-self: flex-start;
	}
}
</style>
