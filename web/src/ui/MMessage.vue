<script setup lang="ts">
import MIcon from "@/ui/MIcon.vue";

// A framed message on the page, such as a failed load with a retry. Errors
// are announced as alerts.
withDefaults(defineProps<{ tone?: "error" | "info" }>(), { tone: "error" });
</script>

<template>
	<div class="m-message" :class="`m-message-${tone}`" :role="tone === 'error' ? 'alert' : 'status'">
		<MIcon class="m-message-icon" :name="tone === 'error' ? 'circle-alert' : 'circle-check'" />
		<div class="m-message-body"><slot /></div>
	</div>
</template>

<style scoped>
.m-message {
	display: flex;
	align-items: flex-start;
	gap: var(--space-3);
	padding: var(--space-4);
	border: 1px solid var(--color-danger-border);
	border-radius: var(--radius-md);
	background: var(--color-danger-soft);
	color: var(--color-danger-strong);
}
.m-message-info {
	border-color: var(--color-info-border);
	background: var(--color-info-soft);
	color: var(--color-info-strong);
}
.m-message-icon {
	margin-top: 0.125rem;
}
.m-message-body {
	flex: 1;
	min-width: 0;
}
</style>
