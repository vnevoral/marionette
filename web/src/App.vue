<script setup lang="ts">
import { RouterView } from "vue-router";
import ConfirmDialog from "primevue/confirmdialog";
import AppShell from "@/components/AppShell.vue";
</script>

<template>
	<AppShell>
		<RouterView />
	</AppShell>
	<ConfirmDialog />
</template>
