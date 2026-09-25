<script setup lang="ts">
import Tag from "primevue/tag";

withDefaults(
	defineProps<{
		label: string;
		icon: string;
		tone: "healthy" | "problem" | "unknown" | "info" | "warning";
	}>(),
	{ tone: "unknown" },
);
</script>

<template>
	<Tag :class="`status-badge status-${tone}`" severity="secondary">
		<i :class="icon" aria-hidden="true" />
		<span>{{ label }}</span>
	</Tag>
</template>

<style scoped>
.status-badge {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	border: 1px solid transparent;
	font-size: 0.78rem;
	font-weight: var(--font-weight-medium);
	letter-spacing: 0;
	line-height: 1.2;
	white-space: nowrap;
}

.status-badge :deep(.p-tag-value) {
	font-weight: var(--font-weight-medium);
}

.status-badge :deep(i) {
	font-size: 0.9em;
}

.status-healthy {
	border-color: var(--color-success-strong);
	background: var(--color-success-strong);
	color: var(--color-on-accent);
}

.status-problem {
	border-color: var(--color-danger-strong);
	background: var(--color-danger-strong);
	color: var(--color-on-accent);
}

.status-unknown {
	border-color: var(--color-border-strong);
	background: var(--color-surface-raised);
	color: var(--color-ink);
}

.status-info {
	border-color: var(--color-info);
	background: var(--color-info);
	color: var(--color-on-accent);
}

.status-warning {
	border-color: var(--color-warning);
	background: var(--color-warning-soft);
	color: var(--color-warning-strong);
}
</style>
