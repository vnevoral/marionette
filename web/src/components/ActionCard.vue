<script setup lang="ts">
import { computed } from "vue";
import { RouterLink } from "vue-router";
import Card from "primevue/card";
import type { ActionCard, StatusSnapshot } from "@/api";
import ActionControls from "@/components/ActionControls.vue";
import RequestState from "@/components/RequestState.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import type { PendingRequest, RequestResult } from "@/types";
import { DEFAULT_CARD_ICON } from "@/ui/icons";
import { lastCheckedLabel } from "@/ui/format";
import {
	ACTIONS,
	EMPTY,
	FEEDBACK,
	NO_STATUS_CHECK,
	REQUEST,
	STATUS,
	statusPresentation,
} from "@/ui/vocabulary";

// Dashboard card (UX spec §5.2): icon and badge, name, description, last
// check, feedback line and the two action buttons.
const props = defineProps<{
	card: ActionCard;
	status?: StatusSnapshot | null;
	pending?: PendingRequest | null;
	result?: RequestResult | null;
	/** The last status read failed; the shown state may be stale (spec §4). */
	statusUnavailable?: boolean;
}>();

const emit = defineEmits<{ run: []; check: [] }>();

const badge = computed(() => {
	if (!props.card.status) return NO_STATUS_CHECK;
	if (props.pending) return REQUEST[props.pending.phase];
	if (props.statusUnavailable) return STATUS.unknown;
	return statusPresentation(props.status?.state);
});

const feedback = computed(() => {
	if (props.pending) return { message: REQUEST[props.pending.phase].label, tone: "info" as const };
	if (props.result) return props.result;
	return { message: "", tone: "info" as const };
});

const detailLink = computed(() => `/cards/${encodeURIComponent(props.card.id)}`);
</script>

<template>
	<Card class="action-card">
		<template #header>
			<div class="card-banner">
				<div class="card-icon" aria-hidden="true">
					<i :class="card.icon || DEFAULT_CARD_ICON" />
				</div>
				<StatusBadge v-bind="badge" />
			</div>
		</template>
		<template #title>
			<span class="card-title">{{ card.name }}</span>
		</template>
		<template #content>
			<p class="card-description">{{ card.description || EMPTY.description }}</p>
			<p v-if="card.status" class="card-checked">{{ lastCheckedLabel(status?.checkedAt) }}</p>
			<p v-if="card.status && statusUnavailable" class="card-unavailable">
				<i class="pi pi-exclamation-circle" aria-hidden="true" />
				<span>{{ FEEDBACK.statusUnavailable }}</span>
			</p>
			<RequestState class="card-feedback" :message="feedback.message" :tone="feedback.tone" />
		</template>
		<template #footer>
			<div class="card-footer">
				<RouterLink class="text-link details-link" :to="detailLink">
					<span>{{ ACTIONS.viewDetails }}</span>
					<i class="pi pi-arrow-up-right" aria-hidden="true" />
				</RouterLink>
				<ActionControls
					:has-status="Boolean(card.status)"
					:pending="pending"
					@run="emit('run')"
					@check="emit('check')"
				/>
			</div>
		</template>
	</Card>
</template>

<style scoped>
.action-card {
	min-width: 0;
	height: 100%;
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
.action-card :deep(.p-card-title) {
	margin: 0;
	font-family: var(--font-display);
	font-size: 1.45rem;
	font-weight: 500;
}
.card-banner {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: var(--space-4) var(--space-4) 0;
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
.card-unavailable {
	display: flex;
	align-items: center;
	gap: var(--space-2);
	margin: var(--space-1) 0 0;
	color: var(--color-warning-strong);
	font-size: 0.85rem;
}
.card-title {
	display: block;
	overflow: hidden;
	color: var(--color-ink);
	text-overflow: ellipsis;
	white-space: nowrap;
}
.card-description {
	display: -webkit-box;
	margin: 0;
	overflow: hidden;
	-webkit-box-orient: vertical;
	-webkit-line-clamp: 3;
	color: var(--color-muted);
	line-height: 1.5;
}
.card-checked {
	margin: var(--space-3) 0 0;
	color: var(--color-muted);
	font-size: 0.85rem;
}
.card-feedback {
	margin-top: var(--space-4);
}
.card-footer {
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
}
.details-link {
	align-self: flex-start;
	font-size: 0.88rem;
}
</style>
