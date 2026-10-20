<script setup lang="ts">
import { ACTIONS } from "@/ui/vocabulary";

// Captured action output on demand (FR-13): collapsed by default, wrapped so
// a long line never scrolls the page, and marked when it was truncated.
defineProps<{ output: string; truncated?: boolean }>();
</script>

<template>
	<details v-if="output" class="run-output">
		<summary>{{ ACTIONS.viewOutput }}</summary>
		<pre>{{ output }}<span v-if="truncated">...</span></pre>
	</details>
</template>

<style scoped>
.run-output summary {
	width: fit-content;
	color: var(--color-info-strong);
	font-size: 0.875rem;
	font-weight: var(--font-weight-medium);
	cursor: pointer;
}
.run-output pre {
	max-width: 100%;
	margin: var(--space-3) 0 0;
	padding: var(--space-3);
	overflow: auto;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	background: var(--color-surface-sunken);
	color: var(--color-ink);
	font-family: var(--font-mono);
	font-size: 0.8125rem;
	white-space: pre-wrap;
}
</style>
