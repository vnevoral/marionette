<script setup lang="ts">
import MIcon from "@/ui/MIcon.vue";
import type { StatusTone } from "@/types";

// A small pill: a status badge (with tone and icon) or a neutral label such
// as "This device".
withDefaults(defineProps<{ label: string; icon?: string; tone?: StatusTone | "neutral" }>(), {
	icon: "",
	tone: "neutral",
});
</script>

<template>
	<span class="m-tag" :class="`m-tag-${tone}`">
		<MIcon v-if="icon" :name="icon" size="0.95em" />
		<span>{{ label }}</span>
	</span>
</template>

<style scoped>
.m-tag {
	display: inline-flex;
	align-items: center;
	gap: 0.375rem;
	min-height: 1.625rem;
	padding: 0 0.625rem;
	border: 1px solid transparent;
	border-radius: 999px;
	font-size: 0.8125rem;
	font-weight: var(--font-weight-semibold);
	line-height: 1.2;
	white-space: nowrap;
}
.m-tag-neutral,
.m-tag-unknown {
	border-color: var(--color-border);
	background: var(--color-surface-sunken);
	color: var(--color-muted-strong);
}
.m-tag-healthy {
	border-color: var(--color-success-border);
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
}
.m-tag-problem {
	border-color: var(--color-danger-border);
	background: var(--color-danger-soft);
	color: var(--color-danger-strong);
}
.m-tag-info {
	border-color: var(--color-info-border);
	background: var(--color-info-soft);
	color: var(--color-info-strong);
}
.m-tag-warning {
	border-color: var(--color-warning-border);
	background: var(--color-warning-soft);
	color: var(--color-warning-strong);
}
</style>
