<script setup lang="ts">
import { computed } from "vue";
import { RouterLink, type RouteLocationRaw } from "vue-router";
import MIcon from "@/ui/MIcon.vue";
import MSpinner from "@/ui/MSpinner.vue";

// The one button of the UI (ADR-0014). With `to` it is a router link that
// looks like a button. Without a label it is an icon button and needs an
// aria-label from the caller. While loading the spinner replaces the icon and
// the button is disabled.
const props = withDefaults(
	defineProps<{
		label?: string;
		icon?: string;
		variant?: "primary" | "secondary" | "ghost" | "danger" | "danger-ghost";
		size?: "md" | "sm";
		type?: "button" | "submit";
		to?: RouteLocationRaw;
		loading?: boolean;
		disabled?: boolean;
	}>(),
	{
		label: "",
		icon: "",
		variant: "secondary",
		size: "md",
		type: "button",
		to: undefined,
		loading: false,
		disabled: false,
	},
);

const classes = computed(() => [
	"m-button",
	`m-button-${props.variant}`,
	`m-button-${props.size}`,
	{ "m-button-icon-only": !props.label, "m-button-loading": props.loading },
]);
const inactive = computed(() => props.disabled || props.loading);
</script>

<template>
	<RouterLink v-if="to" :to="to" :class="classes">
		<MIcon v-if="icon" :name="icon" />
		<span v-if="label" class="m-button-label">{{ label }}</span>
	</RouterLink>
	<button
		v-else
		:type="type"
		:class="classes"
		:disabled="inactive"
		:aria-busy="loading || undefined"
	>
		<MSpinner v-if="loading" class="m-button-spinner" />
		<MIcon v-else-if="icon" :name="icon" />
		<span v-if="label" class="m-button-label">{{ label }}</span>
	</button>
</template>

<style scoped>
.m-button {
	--m-button-height: var(--control-height);
	display: inline-flex;
	min-height: var(--m-button-height);
	align-items: center;
	justify-content: center;
	gap: var(--space-2);
	padding: 0 var(--space-4);
	border: 1px solid transparent;
	border-radius: var(--radius-sm);
	background: transparent;
	color: var(--color-ink);
	font: inherit;
	font-size: 0.9375rem;
	font-weight: var(--font-weight-semibold);
	line-height: 1.2;
	text-decoration: none;
	white-space: nowrap;
	cursor: pointer;
	transition:
		background-color 120ms ease,
		border-color 120ms ease,
		box-shadow 120ms ease,
		color 120ms ease;
}
.m-button-sm {
	--m-button-height: var(--control-height-sm);
	padding: 0 var(--space-3);
	font-size: 0.875rem;
}
.m-button-icon-only {
	width: var(--m-button-height);
	padding: 0;
}
.m-button:disabled {
	cursor: not-allowed;
	opacity: 0.55;
}
.m-button-loading:disabled {
	cursor: progress;
	opacity: 0.8;
}
.m-button-spinner {
	color: currentColor;
}

.m-button-primary {
	background: var(--color-accent);
	box-shadow: var(--shadow-xs);
	color: var(--color-on-accent);
}
.m-button-primary:not(:disabled):hover {
	background: var(--color-accent-strong);
}

.m-button-secondary {
	border-color: var(--color-border-strong);
	background: var(--color-surface);
	box-shadow: var(--shadow-xs);
	color: var(--color-ink);
}
.m-button-secondary:not(:disabled):hover {
	border-color: var(--color-muted);
	background: var(--color-surface-sunken);
}

.m-button-ghost {
	color: var(--color-accent-strong);
}
.m-button-ghost.m-button-icon-only {
	color: var(--color-muted);
}
.m-button-ghost:not(:disabled):hover {
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
}

.m-button-danger {
	background: var(--color-danger);
	box-shadow: var(--shadow-xs);
	color: var(--color-on-accent);
}
.m-button-danger:not(:disabled):hover {
	background: var(--color-danger-strong);
}

.m-button-danger-ghost {
	color: var(--color-danger-strong);
}
.m-button-danger-ghost:not(:disabled):hover {
	background: var(--color-danger-soft);
}
</style>
