<script setup lang="ts">
import MIcon from "@/ui/MIcon.vue";

// The icon tile in the card detail header. The card colour (FR-10a) is shown
// on the Current status panel instead (block 0062).
defineProps<{ icon?: string }>();
</script>

<template>
	<div class="identity-tile" aria-hidden="true">
		<MIcon :name="icon" />
	</div>
</template>

<style scoped>
.identity-tile {
	display: grid;
	width: 64px;
	height: 64px;
	flex-shrink: 0;
	place-items: center;
	border-radius: var(--radius-lg);
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
	font-size: 1.75rem;
}
</style>
