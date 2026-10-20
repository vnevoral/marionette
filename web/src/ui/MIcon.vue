<script setup lang="ts">
import { computed } from "vue";
import { iconComponent } from "@/ui/icons";

// One icon by name (ui/icons.ts). Icons are decorative: the text next to
// them carries the meaning (FR-25), so they are always hidden from assistive
// technology. The spinner icon always turns.
const props = withDefaults(
	defineProps<{ name?: string; size?: number | string; spin?: boolean }>(),
	{
		name: "",
		size: "1.125em",
		spin: false,
	},
);

const icon = computed(() => iconComponent(props.name));
// CSS size, so an em value follows the surrounding text.
const dimension = computed(() => (typeof props.size === "number" ? `${props.size}px` : props.size));
const spinning = computed(() => props.spin || props.name === "loader-circle");
</script>

<template>
	<component
		:is="icon"
		class="m-icon"
		:class="{ 'm-icon-spin': spinning }"
		:style="{ width: dimension, height: dimension }"
		:stroke-width="2"
		aria-hidden="true"
		focusable="false"
	/>
</template>

<style scoped>
.m-icon {
	flex: none;
	vertical-align: -0.15em;
}
.m-icon-spin {
	animation: m-icon-spin 0.9s linear infinite;
}
@keyframes m-icon-spin {
	to {
		transform: rotate(360deg);
	}
}
@media (prefers-reduced-motion: reduce) {
	.m-icon-spin {
		animation-duration: 2.4s;
	}
}
</style>
