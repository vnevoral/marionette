<script setup lang="ts">
import { SwitchRoot, SwitchThumb } from "reka-ui";

// An on/off switch (Reka UI, role="switch"); label it with <label for="id">.
defineProps<{ id?: string }>();
const model = defineModel<boolean>({ default: false });
</script>

<template>
	<SwitchRoot :id="id" v-model="model" class="m-switch">
		<SwitchThumb class="m-switch-thumb" />
	</SwitchRoot>
</template>

<style scoped>
.m-switch {
	position: relative;
	display: inline-flex;
	width: 2.75rem;
	height: 1.5rem;
	flex: none;
	align-items: center;
	padding: 2px;
	border: 1px solid var(--color-border-strong);
	border-radius: 999px;
	background: #d7dfdc;
	cursor: pointer;
	transition:
		background-color 120ms ease,
		border-color 120ms ease;
}
.m-switch:hover {
	background: #c9d4d0;
}
.m-switch[data-state="checked"] {
	border-color: var(--color-accent);
	background: var(--color-accent);
}
.m-switch[data-state="checked"]:hover {
	background: var(--color-accent-strong);
}
.m-switch-thumb {
	display: block;
	width: 1.125rem;
	height: 1.125rem;
	border-radius: 50%;
	background: var(--color-surface);
	box-shadow: var(--shadow-xs);
	transition: transform 120ms ease;
}
.m-switch-thumb[data-state="checked"] {
	transform: translateX(1.25rem);
}
</style>
