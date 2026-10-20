<script setup lang="ts">
import { RouterView } from "vue-router";
import AppShell from "@/components/AppShell.vue";
import MConfirmHost from "@/ui/MConfirmHost.vue";
import MToastHost from "@/ui/MToastHost.vue";
</script>

<template>
	<AppShell>
		<RouterView />
	</AppShell>
	<MConfirmHost />
	<MToastHost />
</template>
