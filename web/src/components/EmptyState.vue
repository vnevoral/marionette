<script setup lang="ts">
defineProps<{ icon: string; title: string; body: string }>();
</script>

<template>
	<section class="empty-state" :aria-label="title">
		<div class="empty-icon" aria-hidden="true"><i :class="icon" /></div>
		<h2>{{ title }}</h2>
		<p>{{ body }}</p>
		<slot name="action" />
	</section>
</template>

<style scoped>
.empty-state {
	display: flex;
	min-height: 320px;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: var(--space-3);
	padding: var(--space-8);
	text-align: center;
}
.empty-state p {
	max-width: 34rem;
	margin: 0;
	color: var(--color-muted);
}
.empty-icon {
	display: grid;
	width: 56px;
	height: 56px;
	place-items: center;
	border-radius: 50%;
	background: var(--color-accent-soft);
	color: var(--color-accent);
	font-size: 1.5rem;
}
</style>
