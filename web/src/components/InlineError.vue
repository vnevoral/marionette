<script setup lang="ts">
import MIcon from "@/ui/MIcon.vue";

defineProps<{ message: string }>();
</script>

<template>
	<p v-if="message" class="inline-error" role="alert">
		<MIcon name="circle-alert" />
		<span>{{ message }}</span>
	</p>
</template>

<style scoped>
.inline-error {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	margin: 0;
	color: var(--color-danger-strong);
	font-size: 0.9rem;
}
</style>
