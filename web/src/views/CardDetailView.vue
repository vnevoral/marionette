<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RouterLink, useRoute } from "vue-router";
import Button from "primevue/button";
import Message from "primevue/message";
import ProgressSpinner from "primevue/progressspinner";
import Tag from "primevue/tag";
import {
	enqueuePrimary,
	enqueueStatus,
	getCard,
	getRuns,
	getStatus,
	getStatusHistory,
	type ActionCard,
	type Run,
	type StatusChange,
	type StatusSnapshot,
} from "@/api";

const route = useRoute();
const card = ref<ActionCard>();
const status = ref<StatusSnapshot>();
const runs = ref<Run[]>([]);
const history = ref<StatusChange[]>([]);
const loading = ref(true);
const notFound = ref(false);
const error = ref("");
const sectionErrors = ref<Record<string, string>>({});
const requestMessage = ref("");
const actionLoading = ref<"primary" | "status" | "">("");

const cardID = computed(() => String(route.params.id));

function formatDuration(durationNanoseconds: number) {
	const milliseconds = durationNanoseconds / 1_000_000;
	if (milliseconds < 1000) return `${Math.round(milliseconds)} ms`;
	return `${(milliseconds / 1000).toFixed(2)} s`;
}

function formatDate(value?: string) {
	if (!value || value.startsWith("0001-01-01")) return "Not available";
	return new Date(value).toLocaleString();
}

function stateView(state = "unknown") {
	if (state === "ok") return { label: "Healthy", icon: "pi pi-check-circle", severity: "success" };
	if (state === "fail")
		return { label: "Problem", icon: "pi pi-exclamation-triangle", severity: "danger" };
	return { label: "Unknown", icon: "pi pi-question-circle", severity: "secondary" };
}

function outcomeLabel(outcome: Run["outcome"]) {
	if (outcome === "ok") return "Successful";
	if (outcome === "timeout") return "Timed out";
	return "Failed";
}

function outcomeSeverity(outcome: Run["outcome"]) {
	if (outcome === "ok") return "success";
	if (outcome === "timeout") return "warn";
	return "danger";
}

async function loadDetail() {
	loading.value = true;
	error.value = "";
	notFound.value = false;
	sectionErrors.value = {};
	try {
		card.value = await getCard(cardID.value);
	} catch (loadError) {
		notFound.value = true;
		error.value = loadError instanceof Error ? loadError.message : "Unable to load card";
		loading.value = false;
		return;
	}

	const results = await Promise.allSettled([
		getStatus(cardID.value),
		getRuns(cardID.value),
		getStatusHistory(cardID.value),
	]);
	const [statusResult, runsResult, historyResult] = results;
	if (statusResult.status === "fulfilled") status.value = statusResult.value;
	else sectionErrors.value.status = "Status data is unavailable";
	if (runsResult.status === "fulfilled") runs.value = runsResult.value ?? [];
	else sectionErrors.value.runs = "Run history is unavailable";
	if (historyResult.status === "fulfilled") history.value = historyResult.value ?? [];
	else sectionErrors.value.history = "Status history is unavailable";
	loading.value = false;
}

async function runAction(action: "primary" | "status") {
	if (action === "status" && !card.value?.status) return;
	actionLoading.value = action;
	requestMessage.value = "Queued";
	try {
		if (action === "primary") await enqueuePrimary(cardID.value);
		else await enqueueStatus(cardID.value);
		requestMessage.value = "Action queued";
		if (action === "status") {
			status.value = await getStatus(cardID.value);
		}
	} catch (actionError) {
		requestMessage.value =
			actionError instanceof Error ? actionError.message : "Unable to queue action";
	} finally {
		actionLoading.value = "";
	}
}

onMounted(loadDetail);
</script>

<template>
	<main class="detail-page">
		<div class="detail-back">
			<RouterLink to="/">
				<i class="pi pi-arrow-left" aria-hidden="true"></i>
				<span>Back to overview</span>
			</RouterLink>
		</div>

		<div v-if="loading" class="loading-state" role="status" aria-live="polite">
			<ProgressSpinner aria-label="Loading card detail" />
			<span>Loading card detail...</span>
		</div>

		<Message v-else-if="notFound" severity="error" :closable="false">
			<h1>Card unavailable</h1>
			<p>{{ error }}</p>
			<RouterLink to="/">Return to overview</RouterLink>
		</Message>

		<template v-else-if="card">
			<header
				class="detail-header flex flex-column md:flex-row align-items-start md:align-items-end justify-content-between gap-6 mb-8"
			>
				<div class="detail-identity flex align-items-center gap-4">
					<div class="detail-icon" aria-hidden="true">{{ card.icon || "◈" }}</div>
					<div>
						<p class="eyebrow">CARD DETAIL</p>
						<h1>{{ card.name }}</h1>
						<p>{{ card.description || "No description provided." }}</p>
					</div>
				</div>
				<div class="detail-actions flex align-items-center gap-4">
					<Tag :severity="stateView(status?.state).severity">
						<i :class="stateView(status?.state).icon" aria-hidden="true"></i>
						<span>{{ stateView(status?.state).label }}</span>
					</Tag>
					<RouterLink class="edit-link" :to="`/manage`">Edit card</RouterLink>
				</div>
			</header>

			<p v-if="requestMessage" class="request-feedback" role="status" aria-live="polite">
				{{ requestMessage }}
			</p>

			<section class="detail-grid grid" aria-label="Card summary">
				<div class="detail-panel summary-panel col-12 md:col-6 p-6">
					<div class="panel-heading">
						<h2>Current status</h2>
						<i class="pi pi-heart" aria-hidden="true"></i>
					</div>
					<p class="summary-state">{{ stateView(status?.state).label }}</p>
					<dl class="summary-list">
						<div>
							<dt>Last checked</dt>
							<dd>{{ formatDate(status?.checkedAt) }}</dd>
						</div>
						<div>
							<dt>Last outcome</dt>
							<dd>
								{{
									status?.lastCheck?.outcome
										? outcomeLabel(status.lastCheck.outcome)
										: "Not checked"
								}}
							</dd>
						</div>
					</dl>
					<p v-if="sectionErrors.status" class="section-error">{{ sectionErrors.status }}</p>
				</div>

				<div class="detail-panel action-panel col-12 md:col-6 p-6">
					<div class="panel-heading">
						<h2>Actions</h2>
						<i class="pi pi-bolt" aria-hidden="true"></i>
					</div>
					<div class="detail-action-buttons flex flex-wrap gap-3">
						<Button
							label="Run action"
							icon="pi pi-play"
							:loading="actionLoading === 'primary'"
							:disabled="Boolean(actionLoading)"
							@click="runAction('primary')"
						/>
						<Button
							v-if="card.status"
							label="Check status"
							icon="pi pi-heart"
							severity="secondary"
							outlined
							:loading="actionLoading === 'status'"
							:disabled="Boolean(actionLoading)"
							@click="runAction('status')"
						/>
					</div>
					<p class="action-hint">
						Actions are queued asynchronously and may take a moment to report a new status.
					</p>
				</div>
			</section>

			<section class="detail-panel mt-4 p-6" aria-labelledby="runs-title">
				<div class="panel-heading">
					<h2 id="runs-title">Recent runs</h2>
					<span class="panel-count">{{ runs.length }}</span>
				</div>
				<p v-if="sectionErrors.runs" class="section-error">{{ sectionErrors.runs }}</p>
				<p v-else-if="!runs.length" class="empty-copy">No primary runs recorded yet.</p>
				<div v-else class="run-list flex flex-column gap-3">
					<article
						v-for="(run, index) in runs"
						:key="`${run.startedAt}-${index}`"
						class="run-row flex flex-column gap-2"
					>
						<div class="run-main">
							<Tag
								:value="outcomeLabel(run.outcome)"
								:severity="outcomeSeverity(run.outcome)"
							/><strong>{{ formatDate(run.startedAt) }}</strong
							><span>{{ formatDuration(run.duration) }}</span>
						</div>
						<small>Exit code {{ run.exitCode }}</small>
						<details v-if="run.output">
							<summary>View output</summary>
							<pre>{{ run.output }}<span v-if="run.truncated">...</span></pre>
						</details>
					</article>
				</div>
			</section>

			<section class="detail-panel mt-4 p-6" aria-labelledby="history-title">
				<div class="panel-heading">
					<h2 id="history-title">Status history</h2>
					<span class="panel-count">{{ history.length }}</span>
				</div>
				<p v-if="sectionErrors.history" class="section-error">{{ sectionErrors.history }}</p>
				<p v-else-if="!history.length" class="empty-copy">No status transitions recorded yet.</p>
				<div v-else class="timeline flex flex-column gap-3">
					<article
						v-for="(change, index) in history"
						:key="`${change.startedAt}-${index}`"
						class="timeline-row flex align-items-start gap-3"
					>
						<span class="timeline-dot" :class="`state-${change.state}`" aria-hidden="true"></span>
						<div>
							<strong>{{ stateView(change.state).label }}</strong>
							<p>{{ formatDate(change.startedAt) }} · {{ formatDuration(change.duration) }}</p>
						</div>
					</article>
				</div>
			</section>
		</template>
	</main>
</template>

<style scoped>
.detail-page {
	max-width: 1200px;
	min-height: calc(100vh - 72px);
	margin: 0 auto;
	padding: clamp(var(--space-6), 5vw, 64px) clamp(var(--space-4), 5vw, 64px);
	color: var(--color-ink);
}
.detail-back {
	margin-bottom: var(--space-6);
}
.detail-back a,
.edit-link {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	color: var(--color-accent-strong);
	font-weight: 700;
	text-decoration: none;
}
.detail-identity {
	min-width: 0;
}
.detail-icon {
	display: grid;
	width: 64px;
	height: 64px;
	flex-shrink: 0;
	place-items: center;
	border-radius: var(--radius-md);
	background: var(--color-accent-soft);
	color: var(--color-accent);
	font-size: 1.6rem;
}
.eyebrow {
	margin: 0 0 var(--space-2);
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
	font-size: clamp(2.25rem, 5vw, 4rem);
	line-height: 1;
}
.detail-identity p:last-child {
	margin: var(--space-2) 0 0;
	color: var(--color-muted);
}
.request-feedback {
	margin: 0 0 var(--space-6);
	padding: var(--space-3) var(--space-4);
	border-left: 3px solid var(--color-info);
	background: var(--color-info-soft);
	color: var(--color-info);
	font-weight: 700;
}
.detail-panel {
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	box-shadow: var(--shadow-subtle);
}
.panel-heading {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: var(--space-3);
	margin-bottom: var(--space-4);
}
.panel-heading h2 {
	font-size: 1.45rem;
}
.panel-heading > i {
	color: var(--color-accent);
}
.panel-count {
	display: grid;
	min-width: 28px;
	height: 28px;
	place-items: center;
	border-radius: 50%;
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
	font-size: 0.8rem;
	font-weight: 800;
}
.summary-state {
	margin: 0 0 var(--space-4);
	color: var(--color-accent-strong);
	font-family: var(--font-display);
	font-size: 2rem;
}
.summary-list {
	display: grid;
	gap: var(--space-3);
	margin: 0;
}
.summary-list div {
	display: flex;
	justify-content: space-between;
	gap: var(--space-4);
	border-top: 1px solid var(--color-border);
	padding-top: var(--space-3);
}
dt {
	color: var(--color-muted);
}
dd {
	margin: 0;
	text-align: right;
}
.action-hint,
.empty-copy {
	margin: var(--space-4) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
.section-error {
	margin: 0;
	color: var(--color-danger);
}
.run-row {
	padding: var(--space-4) 0;
	border-top: 1px solid var(--color-border);
}
.run-main {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: var(--space-3);
}
.run-row small {
	color: var(--color-muted);
}
.run-row details {
	color: var(--color-info);
}
.run-row pre {
	max-width: 100%;
	margin: var(--space-3) 0 0;
	padding: var(--space-3);
	overflow: auto;
	background: var(--color-canvas);
	color: var(--color-ink);
	font-family: monospace;
	white-space: pre-wrap;
}
.timeline-row {
	padding: var(--space-3) 0;
	border-top: 1px solid var(--color-border);
}
.timeline-dot {
	width: 12px;
	height: 12px;
	flex-shrink: 0;
	margin-top: 5px;
	border-radius: 50%;
	background: var(--color-muted);
	box-shadow: 0 0 0 4px var(--color-canvas);
}
.timeline-dot.state-ok {
	background: var(--color-success);
}
.timeline-dot.state-fail {
	background: var(--color-danger);
}
.timeline-row p {
	margin: var(--space-1) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
.loading-state {
	display: grid;
	place-items: center;
	gap: var(--space-3);
	min-height: 320px;
	color: var(--color-muted);
}
</style>
