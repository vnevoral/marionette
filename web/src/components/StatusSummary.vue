<script setup lang="ts">
import { computed } from "vue";
import type { StatusChange, StatusSnapshot } from "@/api";
import { formatDate, transitionDuration } from "@/ui/format";
import { EMPTY, outcomePresentation } from "@/ui/vocabulary";

// Summary of the current device state (UX spec §6): last check, last outcome
// and how long the card has been in this state (FR-17).
const props = defineProps<{
	hasStatus: boolean;
	status?: StatusSnapshot | null;
	currentTransition?: StatusChange;
	now?: number;
}>();

const lastOutcome = computed(() =>
	props.status?.lastCheck
		? outcomePresentation(props.status.lastCheck.outcome).label
		: "Not checked",
);
</script>

<template>
	<dl v-if="hasStatus" class="summary-list">
		<div>
			<dt>Last checked</dt>
			<dd>{{ formatDate(status?.checkedAt) }}</dd>
		</div>
		<div>
			<dt>Last outcome</dt>
			<dd>{{ lastOutcome }}</dd>
		</div>
		<div v-if="currentTransition">
			<dt>In this state for</dt>
			<dd>{{ transitionDuration(currentTransition, now) }}</dd>
		</div>
	</dl>
	<p v-else class="muted-copy">{{ EMPTY.statusNotConfigured }}</p>
</template>

<style scoped>
.summary-list {
	display: grid;
	gap: var(--space-3);
	margin: 0;
}
.summary-list div {
	display: flex;
	justify-content: space-between;
	gap: var(--space-4);
	padding-top: var(--space-3);
	border-top: 1px solid var(--color-border);
}
dt {
	color: var(--color-muted);
}
dd {
	margin: 0;
	text-align: right;
}
.muted-copy {
	margin: var(--space-4) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
</style>
