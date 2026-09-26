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
.run-output {
	color: var(--color-info);
}
.run-output pre {
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
