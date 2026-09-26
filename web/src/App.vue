<script setup lang="ts">
import { RouterView } from "vue-router";
import ConfirmDialog from "primevue/confirmdialog";
import Toast from "primevue/toast";
import AppShell from "@/components/AppShell.vue";
</script>

<template>
	<AppShell>
		<RouterView />
	</AppShell>
	<ConfirmDialog />
	<!-- Page-level notices (useNotify); full width with the page gutter on phones. -->
	<Toast
		position="top-right"
		:breakpoints="{ '640px': { width: 'calc(100vw - 32px)', right: '16px', left: '16px' } }"
	/>
</template>
