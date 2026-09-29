<script setup lang="ts">
import { RouterLink } from "vue-router";

defineProps<{
	title: string;
	eyebrow?: string;
	lede?: string;
	backTo?: string;
	backLabel?: string;
}>();
</script>

<template>
	<header class="page-header">
		<RouterLink v-if="backTo" class="text-link back-link" :to="backTo">
			<i class="pi pi-arrow-left" aria-hidden="true" />
			<span>{{ backLabel }}</span>
		</RouterLink>
		<div class="page-header-row">
			<div class="page-header-identity">
				<slot name="identity" />
				<div>
					<p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
					<h1>{{ title }}</h1>
					<p v-if="lede || $slots.lede" class="page-lede">
						<slot name="lede">{{ lede }}</slot>
					</p>
				</div>
			</div>
			<div v-if="$slots.actions" class="page-header-actions">
				<slot name="actions" />
			</div>
		</div>
	</header>
</template>

<style scoped>
.page-header {
	display: flex;
	flex-direction: column;
	gap: var(--space-6);
	margin-bottom: var(--space-8);
}
.page-header-row {
	display: flex;
	flex-wrap: wrap;
	align-items: flex-end;
	justify-content: space-between;
	gap: var(--space-6);
}
.page-header-identity {
	display: flex;
	min-width: 0;
	align-items: center;
	gap: var(--space-4);
}
.page-header-actions {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: var(--space-3);
}
/* A button label never breaks into two lines (block 0063). */
.page-header-actions > :deep(*) {
	white-space: nowrap;
}
@media (max-width: 48rem) {
	/* The actions get their own full-width row and share it equally. */
	.page-header-row {
		flex-direction: column;
		align-items: stretch;
	}
	.page-header-actions > :deep(*) {
		flex: 1 1 0;
		justify-content: center;
	}
}
@media (max-width: 24rem) {
	.page-header-actions {
		flex-direction: column;
		align-items: stretch;
	}
}
</style>
