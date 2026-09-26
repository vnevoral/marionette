<script setup lang="ts">
import { RouterLink } from "vue-router";
import Button from "primevue/button";
import Message from "primevue/message";
import { ACTIONS } from "@/ui/vocabulary";

// Card detail failure states: a missing card (404) is final, anything else
// can be retried without leaving the page.
defineProps<{ cardID: string; notFound: boolean; error: string }>();
const emit = defineEmits<{ retry: [] }>();
</script>

<template>
	<Message v-if="notFound" severity="error" :closable="false">
		<h1>Card not found</h1>
		<p>There is no card with the id "{{ cardID }}". It may have been deleted.</p>
		<RouterLink to="/">{{ ACTIONS.returnToOverview }}</RouterLink>
	</Message>
	<Message v-else severity="error" :closable="false">
		<h1>Card unavailable</h1>
		<p>{{ error }}</p>
		<div class="flex align-items-center gap-3">
			<Button :label="ACTIONS.tryAgain" text size="small" @click="emit('retry')" />
			<RouterLink to="/">{{ ACTIONS.returnToOverview }}</RouterLink>
		</div>
	</Message>
</template>
