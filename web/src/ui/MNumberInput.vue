<script setup lang="ts">
// A whole-number field: a native number input, so the platform supplies the
// numeric keyboard and the stepper. An empty field is undefined.
withDefaults(defineProps<{ invalid?: boolean; min?: number }>(), {
	invalid: false,
	min: undefined,
});
const model = defineModel<number | undefined>();

function update(event: Event) {
	const text = (event.target as HTMLInputElement).value;
	const value = Number(text);
	model.value = text === "" || !Number.isFinite(value) ? undefined : value;
}
</script>

<template>
	<input
		type="number"
		inputmode="numeric"
		step="1"
		class="m-control m-number-input"
		:class="{ 'm-control-invalid': invalid }"
		:min="min"
		:value="model ?? ''"
		:aria-invalid="invalid || undefined"
		@input="update"
	/>
</template>
