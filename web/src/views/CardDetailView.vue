<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import Button from "primevue/button";
import ProgressSpinner from "primevue/progressspinner";
import { useConfirm } from "primevue/useconfirm";
import {
	ApiError,
	enqueuePrimary,
	enqueueStatus,
	deleteCard,
	getCard,
	type ActionCard,
} from "@/api";
import ActionControls from "@/components/ActionControls.vue";
import DetailErrorState from "@/components/DetailErrorState.vue";
import DetailPanel from "@/components/DetailPanel.vue";
import PageHeader from "@/components/PageHeader.vue";
import RequestState from "@/components/RequestState.vue";
import RunTable from "@/components/RunTable.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import StatusSummary from "@/components/StatusSummary.vue";
import StatusTimeline from "@/components/StatusTimeline.vue";
import { useCardActivity } from "@/composables/useCardActivity";
import { useCardStatus } from "@/composables/useCardStatus";
import { useTransientMessage } from "@/composables/useTransientMessage";
import { singleParam } from "@/router/params";
import type { ActionKind, PendingRequest } from "@/types";
import { DEFAULT_CARD_ICON } from "@/ui/icons";
import {
	ACTIONS,
	EMPTY,
	FEEDBACK,
	LOADING,
	NO_STATUS_CHECK,
	REQUEST,
	statusPresentation,
} from "@/ui/vocabulary";

const route = useRoute();
const router = useRouter();
const confirm = useConfirm();
const card = ref<ActionCard>();
const loading = ref(true);
const notFound = ref(false);
const error = ref("");
const { feedback, show: showMessage, clear: clearMessage } = useTransientMessage();
const pending = ref<PendingRequest | null>(null);
const deleting = ref(false);

const defaultFastPollingWindowSeconds = 120;
const cardID = computed(() => singleParam(route.params.id));
const cardStatus = useCardStatus(cardID, () => Boolean(card.value?.status));
const status = cardStatus.snapshot;
const activity = useCardActivity(cardID);
const { runs, history, runsLoading, historyLoading, errors: activityErrors } = activity;
// Bumped on every id change so responses for a previous card are ignored.
let loadGeneration = 0;
let pendingWait: AbortController | undefined;

const badge = computed(() => {
	if (!card.value?.status) return NO_STATUS_CHECK;
	if (pending.value) return REQUEST[pending.value.phase];
	return statusPresentation(status.value?.state);
});
const currentTransition = computed(() => history.value.find((change) => !change.endedAt));

async function loadDetail() {
	const generation = ++loadGeneration;
	pendingWait?.abort();
	pendingWait = undefined;
	pending.value = null;
	clearMessage();
	loading.value = true;
	error.value = "";
	notFound.value = false;
	card.value = undefined;
	activity.reset();
	cardStatus.set(undefined);
	try {
		const loaded = await getCard(cardID.value);
		if (generation !== loadGeneration) return;
		card.value = loaded;
		cardStatus.set(loaded.currentStatus);
	} catch (loadError) {
		if (generation !== loadGeneration) return;
		notFound.value = loadError instanceof ApiError && loadError.isNotFound;
		error.value = loadError instanceof Error ? loadError.message : FEEDBACK.unableToLoadCard;
		loading.value = false;
		return;
	}
	loading.value = false;
	void activity.load();
}

async function runAction(action: ActionKind) {
	if (pending.value) return;
	const current = card.value;
	if (!current || (action === "status" && !current.status)) return;
	const generation = loadGeneration;
	const previousCheckedAt = status.value?.checkedAt;
	pending.value = { action, phase: "queued" };
	showMessage(FEEDBACK.queued, "info", false);
	const controller = new AbortController();
	pendingWait = controller;
	try {
		if (action === "primary") await enqueuePrimary(current.id);
		else await enqueueStatus(current.id);
		if (generation !== loadGeneration) return;
		if (!current.status) {
			showMessage(FEEDBACK.actionAccepted, "success");
			void activity.load();
			return;
		}
		pending.value = { action, phase: "running" };
		showMessage(FEEDBACK.actionQueued, "info", false);
		const result = await cardStatus.waitForNewer(previousCheckedAt, {
			signal: controller.signal,
			maxWaitMs: (current.fastPollingWindowSeconds || defaultFastPollingWindowSeconds) * 1000,
		});
		if (result === "aborted" || generation !== loadGeneration) return;
		if (result === "updated") showMessage(FEEDBACK.statusUpdated, "success");
		else showMessage(FEEDBACK.resultUnavailable, "error");
		void activity.load();
	} catch (actionError) {
		if (generation !== loadGeneration) return;
		showMessage(
			actionError instanceof Error ? actionError.message : FEEDBACK.unableToQueue,
			"error",
		);
	} finally {
		if (pendingWait === controller) pendingWait = undefined;
		if (generation === loadGeneration) pending.value = null;
	}
}

function removeCard() {
	const current = card.value;
	if (!current || deleting.value) return;
	confirm.require({
		header: ACTIONS.deleteCard,
		message: `Delete "${current.name}"? Its runs and status history are removed as well.`,
		icon: "pi pi-exclamation-triangle",
		acceptLabel: ACTIONS.deleteCard,
		rejectLabel: ACTIONS.cancel,
		acceptProps: { severity: "danger" },
		rejectProps: { severity: "secondary", outlined: true },
		defaultFocus: "reject",
		accept: () => void performDelete(current.id),
	});
}

async function performDelete(id: string) {
	deleting.value = true;
	showMessage(FEEDBACK.deleting, "info", false);
	try {
		await deleteCard(id);
		await router.push("/");
	} catch (deleteError) {
		showMessage(
			deleteError instanceof Error ? deleteError.message : FEEDBACK.unableToDelete,
			"error",
		);
	} finally {
		deleting.value = false;
	}
}

watch(cardID, () => void loadDetail(), { immediate: true });

onBeforeUnmount(() => {
	pendingWait?.abort();
	pendingWait = undefined;
});
</script>

<template>
	<main class="page">
		<div v-if="loading" class="loading-state" role="status" aria-live="polite">
			<ProgressSpinner :aria-label="LOADING.detail" />
			<span>{{ LOADING.detail }}</span>
		</div>

		<DetailErrorState
			v-else-if="notFound || error"
			:card-i-d="cardID"
			:not-found="notFound"
			:error="error"
			@retry="loadDetail"
		/>

		<template v-else-if="card">
			<PageHeader
				eyebrow="Card detail"
				:title="card.name"
				:lede="card.description || EMPTY.description"
				back-to="/"
				:back-label="ACTIONS.backToOverview"
			>
				<template #identity>
					<div class="detail-icon" aria-hidden="true">
						<i :class="card.icon || DEFAULT_CARD_ICON" />
					</div>
				</template>
				<template #actions>
					<RouterLink
						class="primary-action-link"
						:to="`/cards/${encodeURIComponent(card.id)}/edit`"
					>
						<i class="pi pi-pencil" aria-hidden="true" />
						<span>{{ ACTIONS.editCard }}</span>
					</RouterLink>
					<Button
						:label="ACTIONS.deleteCard"
						icon="pi pi-trash"
						severity="danger"
						text
						:loading="deleting"
						:disabled="deleting || Boolean(pending)"
						@click="removeCard"
					/>
				</template>
			</PageHeader>

			<RequestState class="detail-feedback" :message="feedback.message" :tone="feedback.tone" />

			<section class="grid" aria-label="Card summary">
				<div class="col-12 md:col-6 p-2">
					<DetailPanel title="Current status">
						<template #badge><StatusBadge v-bind="badge" /></template>
						<StatusSummary
							:has-status="Boolean(card.status)"
							:status="status"
							:current-transition="currentTransition"
						/>
					</DetailPanel>
				</div>
				<div class="col-12 md:col-6 p-2">
					<DetailPanel title="Actions" icon="pi pi-bolt">
						<ActionControls
							:has-status="Boolean(card.status)"
							:pending="pending"
							:disabled="deleting"
							@run="runAction('primary')"
							@check="runAction('status')"
						/>
						<p class="muted-copy">
							Actions are queued asynchronously and may take a moment to report a new status.
						</p>
					</DetailPanel>
				</div>
			</section>

			<DetailPanel class="mt-4" title="Recent runs" heading-id="runs-title" :count="runs.length">
				<RunTable :runs="runs" :loading="runsLoading" :error="activityErrors.runs" />
			</DetailPanel>

			<DetailPanel
				class="mt-4"
				title="Status history"
				heading-id="history-title"
				:count="history.length"
			>
				<StatusTimeline
					:changes="history"
					:loading="historyLoading"
					:error="activityErrors.history"
				/>
			</DetailPanel>
		</template>
	</main>
</template>

<style scoped>
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
.detail-feedback {
	margin-bottom: var(--space-6);
}
.muted-copy {
	margin: var(--space-4) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
</style>
