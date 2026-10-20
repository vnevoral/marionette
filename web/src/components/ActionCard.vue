<script setup lang="ts">
import { computed } from "vue";
import { RouterLink } from "vue-router";
import type { ActionCard, StatusSnapshot } from "@/api";
import ActionControls from "@/components/ActionControls.vue";
import ActionNote from "@/components/ActionNote.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import type { PendingRequest, RequestResult } from "@/types";
import { cardColorValue } from "@/ui/cardColors";
import MCard from "@/ui/MCard.vue";
import MIcon from "@/ui/MIcon.vue";
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
// check (replaced in place by an action outcome the badge cannot show) and
// the two action buttons.
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

const lastChecked = computed(() =>
	props.card.status ? lastCheckedLabel(props.status?.checkedAt) : "",
);

// The colour stripe (FR-10a) along the top edge; see the style below.
const stripe = computed(() => cardColorValue(props.card.color));

const detailLink = computed(() => `/cards/${encodeURIComponent(props.card.id)}`);
</script>

<template>
	<MCard
		as="article"
		class="action-card"
		:class="{ 'has-color': stripe }"
		:data-color="stripe ? card.color : undefined"
		:style="stripe ? { '--card-stripe': stripe } : undefined"
	>
		<div class="card-banner">
			<div class="card-icon">
				<MIcon :name="card.icon" />
			</div>
			<StatusBadge v-bind="badge" />
		</div>
		<div class="card-content">
			<h3 class="card-title">{{ card.name }}</h3>
			<p class="card-description">{{ card.description || EMPTY.description }}</p>
			<ActionNote
				class="card-checked"
				:fallback="lastChecked"
				:pending="pending"
				:result="result"
				single-line
			/>
			<p v-if="card.status && statusUnavailable" class="card-unavailable">
				<MIcon name="circle-alert" />
				<span>{{ FEEDBACK.statusUnavailable }}</span>
			</p>
		</div>
		<div class="card-footer">
			<RouterLink class="text-link details-link" :to="detailLink">
				<span>{{ ACTIONS.viewDetails }}</span>
				<MIcon name="arrow-up-right" />
			</RouterLink>
			<ActionControls
				:has-status="Boolean(card.status)"
				:pending="pending"
				@run="emit('run')"
				@check="emit('check')"
			/>
		</div>
	</MCard>
</template>

<style scoped>
.action-card {
	display: flex;
	min-width: 0;
	height: 100%;
	flex-direction: column;
	gap: var(--space-4);
	padding: var(--space-6);
	transition:
		box-shadow 160ms ease,
		border-color 160ms ease;
}
.action-card:hover {
	border-color: var(--color-border-strong);
	box-shadow: var(--shadow-md);
}
/* The colour stripe is an inset shadow so a coloured card keeps its size. */
.action-card.has-color {
	border-top-color: var(--card-stripe);
	box-shadow:
		inset 0 6px 0 var(--card-stripe),
		var(--shadow-sm);
}
.action-card.has-color:hover {
	box-shadow:
		inset 0 6px 0 var(--card-stripe),
		var(--shadow-md);
}
.card-banner {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: var(--space-3);
}
.card-icon {
	display: grid;
	width: 44px;
	height: 44px;
	flex: none;
	place-items: center;
	border-radius: var(--radius-md);
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
	font-size: 1.25rem;
}
.card-content {
	display: flex;
	flex: 1;
	flex-direction: column;
	gap: var(--space-2);
	min-width: 0;
}
.card-title {
	margin: 0;
	overflow: hidden;
	color: var(--color-ink);
	font-family: var(--font-display);
	font-size: 1.125rem;
	font-weight: var(--font-weight-semibold);
	letter-spacing: -0.01em;
	line-height: 1.35;
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
	font-size: 0.9375rem;
	line-height: 1.5;
}
.card-checked {
	margin: auto 0 0;
	padding-top: var(--space-2);
}
.card-unavailable {
	display: flex;
	align-items: center;
	gap: var(--space-2);
	margin: 0;
	color: var(--color-warning-strong);
	font-size: 0.85rem;
}
.card-footer {
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
	padding-top: var(--space-4);
	border-top: 1px solid var(--color-border);
}
.details-link {
	align-self: flex-start;
	font-size: 0.875rem;
}
</style>
