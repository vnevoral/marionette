<script setup lang="ts">
// A framed work surface (UX spec §8.3). Cards are never nested.
withDefaults(defineProps<{ as?: string }>(), { as: "section" });
</script>

<template>
	<component :is="as" class="panel"><slot /></component>
</template>
