<script setup lang="ts">
import type { Run } from "@/api";
import InlineError from "@/components/InlineError.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import { formatDate, formatDuration } from "@/ui/format";
import { EMPTY, LOADING, outcomePresentation } from "@/ui/vocabulary";

// Recent primary runs in diagnostic order: newest first, output on demand.
withDefaults(defineProps<{ runs: Run[]; loading: boolean; error?: string }>(), { error: "" });
</script>

<template>
	<InlineError v-if="error" :message="error" />
	<p v-else-if="loading" class="section-loading" role="status">{{ LOADING.runs }}</p>
	<p v-else-if="!runs.length" class="empty-copy">{{ EMPTY.runs }}</p>
	<div v-else class="run-list">
		<article v-for="(run, index) in runs" :key="`${run.startedAt}-${index}`" class="run-row">
			<div class="run-main">
				<StatusBadge v-bind="outcomePresentation(run.outcome)" />
				<strong>{{ formatDate(run.startedAt) }}</strong>
				<span>{{ formatDuration(run.duration) }}</span>
			</div>
			<small>Exit code {{ run.exitCode }}</small>
			<details v-if="run.output">
				<summary>View output</summary>
				<pre>{{ run.output }}<span v-if="run.truncated">...</span></pre>
			</details>
		</article>
	</div>
</template>

<style scoped>
.section-loading,
.empty-copy {
	margin: var(--space-4) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
.run-list {
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
}
.run-row {
	display: flex;
	flex-direction: column;
	gap: var(--space-2);
	padding: var(--space-4) 0;
	border-top: 1px solid var(--color-border);
}
.run-main {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
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
</style>
