<script setup lang="ts">
import MIcon from "@/ui/MIcon.vue";

// A loading indicator; with a label it is announced, without one it is
// decorative (a button that is busy says so through aria-busy).
withDefaults(defineProps<{ label?: string; size?: number | string }>(), {
	label: "",
	size: "1.125em",
});
</script>

<template>
	<span
		class="m-spinner"
		:role="label ? 'progressbar' : undefined"
		:aria-label="label || undefined"
		:aria-hidden="label ? undefined : 'true'"
	>
		<MIcon name="loader-circle" :size="size" />
	</span>
</template>

<style scoped>
.m-spinner {
	display: inline-flex;
	color: var(--color-accent);
}
</style>
