<script setup lang="ts">
import MIcon from "@/ui/MIcon.vue";

defineProps<{ icon: string; title: string; body: string }>();
</script>

<template>
	<section class="empty-state" :aria-label="title">
		<div class="empty-icon"><MIcon :name="icon" /></div>
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
.empty-state h2 {
	margin-top: var(--space-2);
}
.empty-icon {
	display: grid;
	width: 64px;
	height: 64px;
	place-items: center;
	border-radius: var(--radius-lg);
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
	font-size: 1.75rem;
}
</style>
