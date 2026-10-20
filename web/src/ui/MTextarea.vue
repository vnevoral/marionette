<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from "vue";

// A multi-line text field; with autoResize it grows with its content instead
// of scrolling.
const props = withDefaults(defineProps<{ invalid?: boolean; autoResize?: boolean }>(), {
	invalid: false,
	autoResize: false,
});
const model = defineModel<string | undefined>({ default: "" });
const area = ref<HTMLTextAreaElement>();

function resize() {
	const element = area.value;
	if (!props.autoResize || !element) return;
	element.style.height = "auto";
	element.style.height = `${element.scrollHeight + 2}px`;
}

watch(model, () => void nextTick(resize));
onMounted(resize);
</script>

<template>
	<textarea
		ref="area"
		v-model="model"
		class="m-control m-textarea"
		:class="{ 'm-control-invalid': invalid, 'm-textarea-auto': autoResize }"
		:aria-invalid="invalid || undefined"
	/>
</template>
