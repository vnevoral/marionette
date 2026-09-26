<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import Button from "primevue/button";
import Message from "primevue/message";
import ProgressSpinner from "primevue/progressspinner";
import { useConfirm } from "primevue/useconfirm";
import {
	ApiError,
	enqueuePrimary,
	enqueueStatus,
	deleteCard,
	getCard,
	getRuns,
	getStatusHistory,
	type ActionCard,
	type Run,
	type StatusChange,
} from "@/api";
import StatusBadge from "@/components/StatusBadge.vue";
import { useCardStatus } from "@/composables/useCardStatus";
import { singleParam } from "@/router/params";

const route = useRoute();
const router = useRouter();
const confirm = useConfirm();
const card = ref<ActionCard>();
const runs = ref<Run[]>([]);
const history = ref<StatusChange[]>([]);
const loading = ref(true);
const runsLoading = ref(true);
const historyLoading = ref(true);
const notFound = ref(false);
const error = ref("");
const sectionErrors = ref<Record<string, string>>({});
const requestMessage = ref("");
const actionLoading = ref<"primary" | "status" | "">("");
const deleting = ref(false);
const messageVisibleMs = 4000;
let messageTimer: ReturnType<typeof setTimeout> | undefined;
type StatusTone = "healthy" | "problem" | "unknown" | "info" | "warning";

const defaultFastPollingWindowSeconds = 120;
const cardID = computed(() => singleParam(route.params.id));
const cardStatus = useCardStatus(cardID, () => Boolean(card.value?.status));
const status = cardStatus.snapshot;
// Bumped on every id change so responses for a previous card are ignored.
let loadGeneration = 0;
let pendingWait: AbortController | undefined;

function formatDuration(durationNanoseconds: number) {
	const milliseconds = durationNanoseconds / 1_000_000;
	if (milliseconds < 1000) return `${Math.round(milliseconds)} ms`;
	return `${(milliseconds / 1000).toFixed(2)} s`;
}

function formatDate(value?: string) {
	if (!value) return "Not available";
	return new Date(value).toLocaleString();
}

// Terminal outcomes stay visible briefly; pending notes are replaced by the
// outcome and never expire on their own.
function showMessage(message: string, { transient = true } = {}) {
	clearTimeout(messageTimer);
	messageTimer = undefined;
	requestMessage.value = message;
	if (!transient) return;
	messageTimer = setTimeout(() => {
		if (requestMessage.value === message) requestMessage.value = "";
	}, messageVisibleMs);
}

function stateView(state = "unknown"): {
	label: string;
	icon: string;
	severity: "success" | "danger" | "secondary";
	tone: StatusTone;
} {
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

function currentStatusView() {
	if (!card.value?.status) {
		return { label: "No status check", icon: "pi pi-minus-circle", tone: "unknown" as const };
	}
	if (actionLoading.value) {
		return { label: "Running", icon: "pi pi-spin pi-spinner", tone: "info" as const };
	}
	return stateView(status.value?.state);
}

function outcomeLabel(outcome: Run["outcome"]) {
	if (outcome === "ok") return "Successful";
	if (outcome === "timeout") return "Timed out";
	if (outcome === "canceled") return "Canceled";
	return "Failed";
}

function outcomeTone(outcome: Run["outcome"]): StatusTone {
	if (outcome === "ok") return "healthy";
	if (outcome === "timeout") return "warning";
	if (outcome === "canceled") return "unknown";
	return "problem";
}

function outcomeIcon(outcome: Run["outcome"]) {
	if (outcome === "ok") return "pi pi-check-circle";
	if (outcome === "timeout") return "pi pi-clock";
	if (outcome === "canceled") return "pi pi-ban";
	return "pi pi-times-circle";
}

async function loadRuns(generation: number) {
	const id = cardID.value;
	try {
		const loaded = await getRuns(id);
		if (generation !== loadGeneration) return;
		runs.value = loaded ?? [];
		delete sectionErrors.value.runs;
	} catch {
		if (generation === loadGeneration) sectionErrors.value.runs = "Run history is unavailable";
	} finally {
		if (generation === loadGeneration) runsLoading.value = false;
	}
}

async function loadHistory(generation: number) {
	const id = cardID.value;
	try {
		const loaded = await getStatusHistory(id);
		if (generation !== loadGeneration) return;
		history.value = loaded ?? [];
		delete sectionErrors.value.history;
	} catch {
		if (generation === loadGeneration)
			sectionErrors.value.history = "Status history is unavailable";
	} finally {
		if (generation === loadGeneration) historyLoading.value = false;
	}
}

function loadActivity(generation: number) {
	return Promise.allSettled([loadRuns(generation), loadHistory(generation)]);
}

async function loadDetail() {
	const generation = ++loadGeneration;
	pendingWait?.abort();
	pendingWait = undefined;
	actionLoading.value = "";
	showMessage("", { transient: false });
	loading.value = true;
	runsLoading.value = true;
	historyLoading.value = true;
	error.value = "";
	notFound.value = false;
	sectionErrors.value = {};
	card.value = undefined;
	runs.value = [];
	history.value = [];
	cardStatus.set(undefined);
	try {
		const loaded = await getCard(cardID.value);
		if (generation !== loadGeneration) return;
		card.value = loaded;
		cardStatus.set(loaded.currentStatus);
	} catch (loadError) {
		if (generation !== loadGeneration) return;
		notFound.value = loadError instanceof ApiError && loadError.isNotFound;
		error.value = loadError instanceof Error ? loadError.message : "Unable to load card";
		loading.value = false;
		return;
	}
	loading.value = false;
	void loadActivity(generation);
}

async function runAction(action: "primary" | "status") {
	if (actionLoading.value) return;
	const current = card.value;
	if (!current || (action === "status" && !current.status)) return;
	const generation = loadGeneration;
	const previousCheckedAt = status.value?.checkedAt;
	actionLoading.value = action;
	showMessage("Queued", { transient: false });
	const controller = new AbortController();
	pendingWait = controller;
	try {
		if (action === "primary") await enqueuePrimary(current.id);
		else await enqueueStatus(current.id);
		if (generation !== loadGeneration) return;
		if (!current.status) {
			showMessage("Action accepted");
			void loadActivity(generation);
			return;
		}
		showMessage("Action queued", { transient: false });
		const result = await cardStatus.waitForNewer(previousCheckedAt, {
			signal: controller.signal,
			maxWaitMs: (current.fastPollingWindowSeconds || defaultFastPollingWindowSeconds) * 1000,
		});
		if (result === "aborted" || generation !== loadGeneration) return;
		showMessage(result === "updated" ? "Status updated" : "Result not available yet");
		void loadActivity(generation);
	} catch (actionError) {
		if (generation !== loadGeneration) return;
		showMessage(actionError instanceof Error ? actionError.message : "Unable to queue action");
	} finally {
		if (pendingWait === controller) pendingWait = undefined;
		if (generation === loadGeneration) actionLoading.value = "";
	}
}

function removeCard() {
	const current = card.value;
	if (!current || deleting.value) return;
	confirm.require({
		header: "Delete card",
		message: `Delete "${current.name}"? Its runs and status history are removed as well.`,
		icon: "pi pi-exclamation-triangle",
		acceptLabel: "Delete card",
		rejectLabel: "Cancel",
		acceptProps: { severity: "danger" },
		rejectProps: { severity: "secondary", outlined: true },
		defaultFocus: "reject",
		accept: () => void performDelete(current.id),
	});
}

async function performDelete(id: string) {
	deleting.value = true;
	showMessage("Deleting card...", { transient: false });
	try {
		await deleteCard(id);
		await router.push("/");
	} catch (deleteError) {
		showMessage(deleteError instanceof Error ? deleteError.message : "Unable to delete card");
	} finally {
		deleting.value = false;
	}
}

watch(cardID, () => void loadDetail(), { immediate: true });

onBeforeUnmount(() => {
	pendingWait?.abort();
	pendingWait = undefined;
	clearTimeout(messageTimer);
});
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
			<h1>Card not found</h1>
			<p>There is no card with the id "{{ cardID }}". It may have been deleted.</p>
			<RouterLink to="/">Return to overview</RouterLink>
		</Message>

		<Message v-else-if="error" severity="error" :closable="false">
			<h1>Card unavailable</h1>
			<p>{{ error }}</p>
			<div class="flex align-items-center gap-3">
				<Button label="Try again" text size="small" @click="loadDetail" />
				<RouterLink to="/">Return to overview</RouterLink>
			</div>
		</Message>

		<template v-else-if="card">
			<header
				class="detail-header flex flex-column md:flex-row align-items-start md:align-items-end justify-content-between gap-6 mb-8"
			>
				<div class="detail-identity flex align-items-center gap-4">
					<div class="detail-icon" aria-hidden="true">
						<i :class="card.icon || 'pi pi-desktop'" />
					</div>
					<div>
						<p class="eyebrow">CARD DETAIL</p>
						<h1>{{ card.name }}</h1>
						<p>{{ card.description || "No description provided." }}</p>
					</div>
				</div>
				<div class="detail-actions flex align-items-center gap-3">
					<RouterLink
						class="edit-button primary-action-button"
						:to="`/cards/${encodeURIComponent(card.id)}/edit`"
					>
						<i class="pi pi-pencil" aria-hidden="true" />
						<span>Edit card</span>
					</RouterLink>
					<Button
						label="Delete card"
						icon="pi pi-trash"
						severity="danger"
						text
						:loading="deleting"
						:disabled="deleting || Boolean(actionLoading)"
						@click="removeCard"
					/>
				</div>
			</header>

			<p v-if="requestMessage" class="request-feedback" role="status" aria-live="polite">
				{{ requestMessage }}
			</p>

			<section class="detail-grid grid" aria-label="Card summary">
				<div class="col-12 md:col-6 p-2">
					<div class="detail-panel summary-panel p-6 h-full">
						<div class="panel-heading current-status-heading">
							<h2>Current status</h2>
							<StatusBadge
								:label="currentStatusView().label"
								:icon="currentStatusView().icon"
								:tone="currentStatusView().tone"
							/>
						</div>
						<dl v-if="card.status" class="summary-list">
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
						<p v-else class="status-empty-copy">
							Status monitoring is not configured for this card.
						</p>
					</div>
				</div>

				<div class="col-12 md:col-6 p-2">
					<div class="detail-panel action-panel p-6 h-full">
						<div class="panel-heading">
							<h2>Actions</h2>
							<i class="pi pi-bolt" aria-hidden="true"></i>
						</div>
						<div class="detail-action-buttons flex flex-wrap gap-3">
							<Button
								label="Run action"
								icon="pi pi-play"
								class="primary-action-button"
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
				</div>
			</section>

			<section class="detail-panel mt-4 p-6" aria-labelledby="runs-title">
				<div class="panel-heading">
					<h2 id="runs-title">Recent runs</h2>
					<span class="panel-count">{{ runs.length }}</span>
				</div>
				<p v-if="sectionErrors.runs" class="section-error">{{ sectionErrors.runs }}</p>
				<div v-else-if="runsLoading" class="section-loading" role="status">
					Loading recent runs...
				</div>
				<p v-else-if="!runs.length" class="empty-copy">No primary runs recorded yet.</p>
				<div v-else class="run-list flex flex-column gap-3">
					<article
						v-for="(run, index) in runs"
						:key="`${run.startedAt}-${index}`"
						class="run-row flex flex-column gap-2"
					>
						<div class="run-main">
							<StatusBadge
								:label="outcomeLabel(run.outcome)"
								:icon="outcomeIcon(run.outcome)"
								:tone="outcomeTone(run.outcome)"
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
				<div v-else-if="historyLoading" class="section-loading" role="status">
					Loading status history...
				</div>
				<p v-else-if="!history.length" class="empty-copy">No status transitions recorded yet.</p>
				<div v-else class="timeline flex flex-column gap-3">
					<article
						v-for="(change, index) in history"
						:key="`${change.startedAt}-${index}`"
						class="timeline-row flex align-items-start gap-3"
					>
						<StatusBadge
							:label="stateView(change.state).label"
							:icon="stateView(change.state).icon"
							:tone="stateView(change.state).tone"
						/>
						<div>
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
	max-width: var(--page-max-width);
	margin: 0 auto;
	padding: var(--page-padding-y) var(--page-padding-x);
	color: var(--color-ink);
}
.detail-back {
	margin-bottom: var(--space-6);
}
.detail-back a,
.edit-link,
.edit-button {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	color: var(--color-accent-strong);
	font-weight: var(--font-weight-medium);
	text-decoration: none;
}

.edit-button {
	min-height: 40px;
	padding: 0 var(--space-4);
	border: 1px solid var(--color-accent);
	border-radius: var(--radius-sm);
}

.edit-button:hover {
	border-color: var(--color-accent-strong);
	background: #d8ebdc;
	color: var(--color-accent-strong);
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
	font-weight: var(--font-weight-semibold);
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
	font-weight: var(--font-weight-medium);
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
	font-weight: var(--font-weight-semibold);
}
.summary-list {
	display: grid;
	gap: var(--space-3);
	margin: 0;
}
.status-empty-copy {
	margin: var(--space-4) 0 0;
	color: var(--color-muted);
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
.section-loading {
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
