<script setup lang="ts">
import { RouterLink } from "vue-router";
import MButton from "@/ui/MButton.vue";
import MMessage from "@/ui/MMessage.vue";
import { ACTIONS } from "@/ui/vocabulary";

// Card detail failure states: a missing card (404) is final, anything else
// can be retried without leaving the page.
defineProps<{ cardID: string; notFound: boolean; error: string }>();
const emit = defineEmits<{ retry: [] }>();
</script>

<template>
	<MMessage v-if="notFound" class="detail-error">
		<h1>Card not found</h1>
		<p>There is no card with the id "{{ cardID }}". It may have been deleted.</p>
		<RouterLink class="text-link" to="/">{{ ACTIONS.returnToOverview }}</RouterLink>
	</MMessage>
	<MMessage v-else class="detail-error">
		<h1>Card unavailable</h1>
		<p>{{ error }}</p>
		<div class="flex align-items-center gap-3">
			<MButton :label="ACTIONS.tryAgain" variant="ghost" size="sm" @click="emit('retry')" />
			<RouterLink class="text-link" to="/">{{ ACTIONS.returnToOverview }}</RouterLink>
		</div>
	</MMessage>
</template>

<style scoped>
.detail-error h1 {
	margin: 0;
	font-size: 1.25rem;
}
.detail-error p {
	margin: var(--space-2) 0 var(--space-3);
	color: var(--color-ink);
}
</style>
