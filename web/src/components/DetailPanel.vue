<script setup lang="ts">
// Framed work surface of the detail view with a heading row: title, optional
// count or decorative icon, and a slot for a badge.
defineProps<{ title: string; headingId?: string; count?: number; icon?: string }>();
</script>

<template>
	<section class="panel detail-panel" :aria-labelledby="headingId">
		<div class="panel-heading">
			<h2 :id="headingId">{{ title }}</h2>
			<slot name="badge">
				<span v-if="count !== undefined" class="panel-count">{{ count }}</span>
				<i v-else-if="icon" :class="icon" class="panel-icon" aria-hidden="true" />
			</slot>
		</div>
		<slot />
	</section>
</template>

<style scoped>
.detail-panel {
	height: 100%;
	padding: var(--space-6);
}
.panel-heading {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: var(--space-3);
	margin-bottom: var(--space-4);
}
.panel-heading h2 {
	font-size: 1.45rem;
}
.panel-icon {
	color: var(--color-accent);
}
.panel-count {
	display: grid;
	min-width: 28px;
	height: 28px;
	place-items: center;
	border-radius: 50%;
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
	font-size: 0.8rem;
	font-weight: var(--font-weight-semibold);
}
</style>
