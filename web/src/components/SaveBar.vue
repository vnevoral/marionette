<script setup lang="ts">
import { RouterLink } from "vue-router";
import MButton from "@/ui/MButton.vue";
import { ACTIONS } from "@/ui/vocabulary";

// Sticky save/cancel bar of the card form (UX spec §7.2).
defineProps<{ saving: boolean; cancelTo: string }>();
const emit = defineEmits<{ save: [] }>();
</script>

<template>
	<div class="save-bar">
		<RouterLink class="text-link" :to="cancelTo">{{ ACTIONS.cancel }}</RouterLink>
		<MButton
			:label="ACTIONS.saveCard"
			icon="check"
			variant="primary"
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
	padding: var(--space-4) var(--space-6);
	border-top: 1px solid var(--color-border);
	border-radius: 0 0 var(--radius-lg) var(--radius-lg);
	background: var(--color-surface);
}
</style>
