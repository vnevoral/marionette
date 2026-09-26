<script setup lang="ts">
import { RouterLink } from "vue-router";
import Button from "primevue/button";
import { ACTIONS } from "@/ui/vocabulary";

// Sticky save/cancel bar of the card form (UX spec §7.2).
defineProps<{ saving: boolean; cancelTo: string }>();
const emit = defineEmits<{ save: [] }>();
</script>

<template>
	<div class="save-bar">
		<RouterLink class="text-link" :to="cancelTo">{{ ACTIONS.cancel }}</RouterLink>
		<Button
			:label="ACTIONS.saveCard"
			icon="pi pi-check"
			class="primary-action-button"
			:loading="saving"
			@click="emit('save')"
		/>
	</div>
</template>

<style scoped>
.save-bar {
	position: sticky;
	bottom: 0;
	z-index: 1;
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	justify-content: flex-end;
	gap: var(--space-3);
	margin: 0 calc(-1 * var(--space-4));
	padding: var(--space-3) var(--space-4);
	border-top: 1px solid var(--color-border);
	background: color-mix(in srgb, var(--color-surface) 94%, transparent);
	backdrop-filter: blur(4px);
}
</style>
