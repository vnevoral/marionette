<script setup lang="ts">
import type { StatusTone } from "@/types";
import MTag from "@/ui/MTag.vue";

// Text, icon and tone always travel together (FR-25); pass a Presentation
// from the vocabulary with v-bind.
withDefaults(
	defineProps<{
		label: string;
		icon: string;
		tone?: StatusTone;
	}>(),
	{ tone: "unknown" },
);
</script>

<template>
	<MTag :class="`status-badge status-${tone}`" :label="label" :icon="icon" :tone="tone" />
</template>
