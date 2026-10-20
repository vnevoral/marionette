<script setup lang="ts">
defineProps<{ title: string; description?: string; headingId: string }>();
</script>

<template>
	<section class="form-section" :aria-labelledby="headingId">
		<div class="section-heading">
			<h2 :id="headingId">{{ title }}</h2>
			<p v-if="description">{{ description }}</p>
		</div>
		<slot />
	</section>
</template>

<style scoped>
.form-section + .form-section {
	margin-top: var(--space-8);
	padding-top: var(--space-6);
	border-top: 1px solid var(--color-border);
}
.section-heading {
	margin-bottom: var(--space-4);
}
.section-heading h2 {
	margin: 0;
}
.section-heading p {
	margin: var(--space-1) 0 0;
	color: var(--color-muted);
}
</style>
