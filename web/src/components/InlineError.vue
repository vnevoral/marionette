<script setup lang="ts">
defineProps<{ message: string }>();
</script>

<template>
	<p v-if="message" class="inline-error" role="alert">
		<i class="pi pi-exclamation-circle" aria-hidden="true" />
		<span>{{ message }}</span>
	</p>
</template>

<style scoped>
.inline-error {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	margin: 0;
	color: var(--color-danger);
	font-size: 0.9rem;
}
</style>
