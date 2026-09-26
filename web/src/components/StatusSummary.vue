<script setup lang="ts">
import { computed } from "vue";
import type { StatusChange, StatusSnapshot } from "@/api";
import RunOutput from "@/components/RunOutput.vue";
import { formatDate, formatDuration, transitionDuration } from "@/ui/format";
import { EMPTY, FEEDBACK, outcomePresentation } from "@/ui/vocabulary";

// Summary of the current device state (UX spec §6): last check, its outcome,
// exit code, duration and output on demand (FR-21a), and how long the card
// has been in this state (FR-17).
const props = defineProps<{
	hasStatus: boolean;
	status?: StatusSnapshot | null;
	currentTransition?: StatusChange;
	now?: number;
	/** The last status read failed; the values below may be stale (spec §4). */
	unavailable?: boolean;
}>();

const lastOutcome = computed(() =>
	props.status?.lastCheck
		? outcomePresentation(props.status.lastCheck.outcome).label
		: "Not checked",
);
</script>

<template>
	<p v-if="hasStatus && unavailable" class="summary-unavailable">
		<i class="pi pi-exclamation-circle" aria-hidden="true" />
		<span>{{ FEEDBACK.statusUnavailable }}</span>
	</p>
	<dl v-if="hasStatus" class="summary-list">
		<div>
			<dt>Last checked</dt>
			<dd>{{ formatDate(status?.checkedAt) }}</dd>
		</div>
		<div>
			<dt>Last outcome</dt>
			<dd>{{ lastOutcome }}</dd>
		</div>
		<template v-if="status?.lastCheck">
			<div>
				<dt>Exit code</dt>
				<dd>{{ status.lastCheck.exitCode }}</dd>
			</div>
			<div>
				<dt>Duration</dt>
				<dd>{{ formatDuration(status.lastCheck.duration) }}</dd>
			</div>
		</template>
		<div v-if="currentTransition">
			<dt>In this state for</dt>
			<dd>{{ transitionDuration(currentTransition, now) }}</dd>
		</div>
	</dl>
	<RunOutput
		v-if="hasStatus && status?.lastCheck"
		class="summary-output"
		:output="status.lastCheck.output"
		:truncated="status.lastCheck.truncated"
	/>
	<p v-if="!hasStatus" class="muted-copy">{{ EMPTY.statusNotConfigured }}</p>
</template>

<style scoped>
.summary-unavailable {
	display: flex;
	align-items: center;
	gap: var(--space-2);
	margin: 0 0 var(--space-3);
	color: var(--color-warning-strong);
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
.summary-output {
	margin-top: var(--space-3);
}
.muted-copy {
	margin: var(--space-4) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
</style>
