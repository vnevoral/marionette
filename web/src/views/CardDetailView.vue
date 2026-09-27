<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import Button from "primevue/button";
import ProgressSpinner from "primevue/progressspinner";
import { useConfirm } from "primevue/useconfirm";
import { ApiError, deleteCard, getCard, type ActionCard } from "@/api";
import ActionControls from "@/components/ActionControls.vue";
import ActionNote from "@/components/ActionNote.vue";
import DetailErrorState from "@/components/DetailErrorState.vue";
import DetailPanel from "@/components/DetailPanel.vue";
import CardIdentityTile from "@/components/CardIdentityTile.vue";
import PageHeader from "@/components/PageHeader.vue";
import RequestState from "@/components/RequestState.vue";
import RunTable from "@/components/RunTable.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import StatusSummary from "@/components/StatusSummary.vue";
import StatusTimeline from "@/components/StatusTimeline.vue";
import { useCardActivity } from "@/composables/useCardActivity";
import { outcomeResult, requestAction } from "@/composables/useActionRequest";
import { useCardStatus } from "@/composables/useCardStatus";
import { useNotify } from "@/composables/useNotify";
import { useTransientMessage, useTransientResult } from "@/composables/useTransientMessage";
import { singleParam } from "@/router/params";
import type { ActionKind, PendingRequest } from "@/types";
import {
	ACTIONS,
	EMPTY,
	FEEDBACK,
	LOADING,
	NO_STATUS_CHECK,
	REQUEST,
	STATUS,
	statusPresentation,
} from "@/ui/vocabulary";

const route = useRoute();
const router = useRouter();
const confirm = useConfirm();
const notify = useNotify();
const card = ref<ActionCard>();
const loading = ref(true);
const notFound = ref(false);
const error = ref("");
const { feedback, show: showMessage, clear: clearMessage } = useTransientMessage();
const pending = ref<PendingRequest | null>(null);
// Outcome of the last action, shown in the Actions panel (UX spec §6); the
// page-level feedback line above the summary is left to deleting the card.
const { result: actionResult, show: showActionResult } = useTransientResult();
const deleting = ref(false);

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
	if (cardStatus.failed.value) return STATUS.unknown;
	return statusPresentation(status.value?.state);
});
const currentTransition = computed(() => history.value.find((change) => !change.endedAt));

async function loadDetail() {
	const generation = ++loadGeneration;
	pendingWait?.abort();
	pendingWait = undefined;
	pending.value = null;
	showActionResult(null);
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
	const controller = new AbortController();
	pendingWait = controller;
	const runBaseline = activity.runBaseline();
	try {
		const outcome = await requestAction(current, action, {
			signal: controller.signal,
			onPhase(phase) {
				pending.value = { action, phase };
				showActionResult(null);
			},
			// The run shows up in Recent runs once the server records it (block 0050).
			onAccepted: () => void (action === "primary" && activity.waitForNewRun(current, runBaseline)),
			onSnapshot: (snapshot) => cardStatus.apply(current.id, snapshot),
			onError: () => cardStatus.fail(current.id),
		});
		if (generation !== loadGeneration) return;
		const result = outcomeResult(outcome, {
			accepted: FEEDBACK.actionAccepted,
			updated: FEEDBACK.statusUpdated,
		});
		if (!result) return;
		showActionResult(result);
		if (outcome.kind !== "failed") void activity.loadHistory();
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
		accept: () => void performDelete(current),
	});
}

async function performDelete(target: ActionCard) {
	deleting.value = true;
	showMessage(FEEDBACK.deleting, "info", false);
	try {
		await deleteCard(target.id);
		// The detail page is gone after this, so the confirmation travels
		// with the navigation as a toast (UX spec §4).
		await router.push("/");
		notify.success(FEEDBACK.cardDeleted, `"${target.name}" and its history were removed.`);
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
					<CardIdentityTile :icon="card.icon" :color="card.color" />
				</template>
				<template #actions>
					<Button v-slot="slotProps" as-child>
						<RouterLink
							:class="[slotProps.class, 'primary-action-button']"
							:to="`/cards/${encodeURIComponent(card.id)}/edit`"
						>
							<i class="pi pi-pencil p-button-icon p-button-icon-left" aria-hidden="true" />
							<span class="p-button-label">{{ ACTIONS.editCard }}</span>
						</RouterLink>
					</Button>
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
							:unavailable="cardStatus.failed.value"
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
						<ActionNote
							class="action-note-slot"
							:fallback="FEEDBACK.actionsAsync"
							:pending="pending"
							:result="actionResult"
						/>
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
.detail-feedback {
	margin-bottom: var(--space-6);
}
.action-note-slot {
	/* Room for the two-line fallback so an outcome never changes the height. */
	min-height: 3em;
	margin: var(--space-4) 0 0;
}
</style>
