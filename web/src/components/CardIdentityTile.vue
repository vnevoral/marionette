<script setup lang="ts">
import { computed } from "vue";
import { cardColorValue } from "@/ui/cardColors";
import { DEFAULT_CARD_ICON } from "@/ui/icons";

// The icon tile in the card detail header. A card colour (FR-10a) is a stripe
// along the tile's top edge, drawn by an inset shadow so the tile keeps its
// size; a card without a colour (or with one outside the palette) has none.
const props = defineProps<{ icon?: string; color?: string }>();

const stripe = computed(() => cardColorValue(props.color));
</script>

<template>
	<div
		class="identity-tile"
		:class="{ 'has-color': stripe }"
		:data-color="stripe ? color : undefined"
		:style="stripe ? { '--card-stripe': stripe } : undefined"
		aria-hidden="true"
	>
		<i :class="icon || DEFAULT_CARD_ICON" />
	</div>
</template>

<style scoped>
.identity-tile {
	display: grid;
	width: 64px;
	height: 64px;
	flex-shrink: 0;
	place-items: center;
	border-radius: var(--radius-md);
	background: var(--color-accent-soft);
	color: var(--color-accent);
	font-size: 1.6rem;
}
.identity-tile.has-color {
	box-shadow: inset 0 4px 0 var(--card-stripe);
}
</style>
