<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { getHealth } from "@/api";
import { useStatusEvents } from "@/composables/useStatusEvents";

// The running version in the footer (FR-41a). It is read once, and again when
// the status stream comes back after a disconnect: that is usually the
// service restarting during an upgrade, so the new version shows without a
// reload. A failed read hides the line and affects nothing else.
const version = ref("");
const events = useStatusEvents();
let disconnected = false;

async function load() {
	try {
		version.value = (await getHealth()).version;
	} catch {
		version.value = "";
	}
}

watch(events.connected, (connected) => {
	if (!connected) disconnected = true;
	else if (disconnected) {
		disconnected = false;
		void load();
	}
});

onMounted(load);
</script>

<template>
	<p v-if="version" class="app-version">Marionette {{ version }}</p>
</template>

<style scoped>
.app-version {
	margin: 0;
	color: var(--color-muted);
	font-size: 0.75rem;
}
</style>
