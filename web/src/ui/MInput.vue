<script setup lang="ts">
// A single-line text field (ADR-0014). Attributes such as id, placeholder,
// maxlength or aria-* go to the input itself, which is the root element, so
// a parent can focus it through the component's $el.
withDefaults(defineProps<{ invalid?: boolean }>(), { invalid: false });
const model = defineModel<string | undefined>({ default: "" });
</script>

<template>
	<input
		v-model="model"
		class="m-control m-input"
		:class="{ 'm-control-invalid': invalid }"
		:aria-invalid="invalid || undefined"
	/>
</template>
