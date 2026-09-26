<script setup lang="ts">
import Button from "primevue/button";
import type { PendingRequest } from "@/types";
import { ACTIONS } from "@/ui/vocabulary";

// Run action / Check status pair. Both are blocked while a request for the
// card is pending; the pending one shows a spinner.
const props = defineProps<{
	hasStatus: boolean;
	pending?: PendingRequest | null;
	disabled?: boolean;
}>();

const emit = defineEmits<{ run: []; check: [] }>();

function blocked() {
	return Boolean(props.pending) || Boolean(props.disabled);
}
</script>

<template>
	<div class="action-controls">
		<Button
			:label="ACTIONS.run"
			icon="pi pi-play"
			class="primary-action-button"
			:loading="pending?.action === 'primary'"
			:disabled="blocked()"
			@click="emit('run')"
		/>
		<Button
			v-if="hasStatus"
			:label="ACTIONS.check"
			icon="pi pi-heart"
			severity="secondary"
			outlined
			:loading="pending?.action === 'status'"
			:disabled="blocked()"
			@click="emit('check')"
		/>
	</div>
</template>

<style scoped>
.action-controls {
	display: flex;
	flex-wrap: wrap;
	gap: var(--space-2);
}
.action-controls :deep(.p-button) {
	flex: 1 1 auto;
}
@media (max-width: 24rem) {
	.action-controls {
		flex-direction: column;
		align-items: stretch;
	}
}
</style>
