<script setup lang="ts">
import { ref, onMounted } from "vue";
import Button from "primevue/button";
import Card from "primevue/card";

const status = ref<string>("checking...");

async function checkHealth() {
	try {
		const res = await fetch("/api/health");
		const data = await res.json();
		status.value = data.status ?? "unknown";
	} catch {
		status.value = "unreachable";
	}
}

onMounted(checkHealth);
</script>

<template>
	<main style="max-width: 40rem; margin: 4rem auto; padding: 0 1rem">
		<Card>
			<template #title>
				Marionette
			</template>
			<template #subtitle>
				Go + Vue (PrimeVue) SPA embedded in a single binary
			</template>
			<template #content>
				<p>
					API health: <strong>{{ status }}</strong>
				</p>
				<Button
					label="Recheck"
					icon="pi pi-refresh"
					@click="checkHealth"
				/>
			</template>
		</Card>
	</main>
</template>
