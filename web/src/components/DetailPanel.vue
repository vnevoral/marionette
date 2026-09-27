<script setup lang="ts">
import { computed } from "vue";
import { cardColorValue } from "@/ui/cardColors";

// Framed work surface of the detail view with a heading row: title, optional
// count or decorative icon, and a slot for a badge. A card colour (FR-10a) is
// a stripe along the top edge, drawn like on the dashboard card so the panel
// keeps its size; a colour outside the palette draws none.
const props = defineProps<{
	title: string;
	headingId?: string;
	count?: number;
	icon?: string;
	color?: string;
}>();

const stripe = computed(() => cardColorValue(props.color));
</script>

<template>
	<section
		class="panel detail-panel"
		:class="{ 'has-color': stripe }"
		:data-color="stripe ? color : undefined"
		:style="stripe ? { '--card-stripe': stripe } : undefined"
		:aria-labelledby="headingId"
	>
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
.detail-panel.has-color {
	border-top-color: var(--card-stripe);
	box-shadow:
		inset 0 5px 0 var(--card-stripe),
		var(--shadow-subtle);
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
