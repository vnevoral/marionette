<script setup lang="ts">
// One-line operation feedback (UX spec §4): info while pending, success or
// error once finished. Renders nothing without a message so the live region
// stays quiet.
withDefaults(defineProps<{ message: string; tone?: "info" | "success" | "error" }>(), {
	tone: "info",
});
</script>

<template>
	<p
		v-if="message"
		class="request-feedback"
		:class="`request-feedback-${tone}`"
		role="status"
		aria-live="polite"
	>
		{{ message }}
	</p>
</template>

<style scoped>
.request-feedback {
	display: inline-flex;
	margin: 0;
	padding: var(--space-2) var(--space-3);
	border-left: 3px solid var(--color-info);
	background: var(--color-info-soft);
	color: var(--color-info);
	font-size: 0.85rem;
	font-weight: var(--font-weight-medium);
}
.request-feedback-success {
	border-left-color: var(--color-success);
	background: var(--color-accent-soft);
	color: var(--color-success-strong);
}
.request-feedback-error {
	border-left-color: var(--color-danger);
	background: var(--color-danger-soft);
	color: var(--color-danger-strong);
}
</style>
