<script setup lang="ts">
import { computed } from "vue";
import { useStatusEvents } from "@/composables/useStatusEvents";

// Keeps the shared status stream open for the whole app and reflects it in
// the header: "Live" while the stream is up, "Reconnecting" while REST
// polling covers for it.
const events = useStatusEvents();
events.subscribe({});

const label = computed(() => (events.connected.value ? "Live" : "Reconnecting"));
const tone = computed(() => (events.connected.value ? "connected" : "disconnected"));
</script>

<template>
	<div class="connection-state" :class="`connection-${tone}`" role="status" aria-live="polite">
		<span class="connection-dot" aria-hidden="true"></span>
		<span>{{ label }}</span>
	</div>
</template>

<style scoped>
.connection-state {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	color: var(--color-muted);
	font-size: 0.8rem;
}

.connection-dot {
	width: 8px;
	height: 8px;
	border-radius: 50%;
	background: var(--color-success);
	box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 15%, transparent);
}

.connection-disconnected .connection-dot {
	background: var(--color-warning);
	box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-warning) 18%, transparent);
}
</style>
