<script setup lang="ts">
import MButton from "@/ui/MButton.vue";
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
		<MButton
			:label="ACTIONS.run"
			icon="play"
			variant="primary"
			:loading="pending?.action === 'primary'"
			:disabled="blocked()"
			@click="emit('run')"
		/>
		<MButton
			v-if="hasStatus"
			:label="ACTIONS.check"
			icon="activity"
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
.action-controls > * {
	flex: 1 1 auto;
}
@media (max-width: 24rem) {
	.action-controls {
		flex-direction: column;
		align-items: stretch;
	}
}
</style>
