<script setup lang="ts" generic="V extends string">
import { computed } from "vue";
import {
	SelectContent,
	SelectIcon,
	SelectItem,
	SelectItemIndicator,
	SelectItemText,
	SelectPortal,
	SelectRoot,
	SelectTrigger,
	SelectValue,
	SelectViewport,
} from "reka-ui";
import MIcon from "@/ui/MIcon.vue";

// A select from a short list of options (Reka UI: listbox semantics, typeahead
// and keyboard). Name it with ariaLabel or ariaLabelledby. The `option` slot
// renders an option, the `value` slot the trigger; both receive the option, or
// undefined for a value that is not among the options (such as an old icon).
const props = withDefaults(
	defineProps<{
		options: readonly { label: string; value: V }[];
		invalid?: boolean;
		ariaLabel?: string;
		ariaLabelledby?: string;
		placeholder?: string;
	}>(),
	{ invalid: false, ariaLabel: undefined, ariaLabelledby: undefined, placeholder: "" },
);
const model = defineModel<V | undefined>();

// Reka UI reserves "" for "no selection", so the empty option (colour None)
// travels under a sentinel and is mapped back on the way out.
const EMPTY = "\u0000empty";
const toItem = (value: string | undefined) => (value === "" ? EMPTY : value);
const fromItem = (value: string) => (value === EMPTY ? "" : value) as V;

const selected = computed(() => props.options.find((option) => option.value === model.value));
const rootValue = computed(() => toItem(model.value ?? undefined));

function update(value: unknown) {
	if (typeof value === "string") model.value = fromItem(value);
}
</script>

<template>
	<SelectRoot :model-value="rootValue" @update:model-value="update">
		<SelectTrigger
			class="m-control m-select"
			:class="{ 'm-control-invalid': invalid }"
			:aria-label="ariaLabel"
			:aria-labelledby="ariaLabelledby"
			:aria-invalid="invalid || undefined"
		>
			<SelectValue class="m-select-value" :placeholder="placeholder">
				<slot name="value" :option="selected" :value="model">
					{{ selected?.label ?? placeholder }}
				</slot>
			</SelectValue>
			<SelectIcon class="m-select-chevron"><MIcon name="chevron-down" /></SelectIcon>
		</SelectTrigger>
		<SelectPortal>
			<SelectContent class="m-select-content" position="popper" :side-offset="6">
				<SelectViewport class="m-select-viewport">
					<SelectItem
						v-for="option in options"
						:key="option.value"
						:value="toItem(option.value)!"
						:text-value="option.label"
						class="m-select-item"
					>
						<SelectItemText>
							<slot name="option" :option="option">{{ option.label }}</slot>
						</SelectItemText>
						<SelectItemIndicator class="m-select-check"><MIcon name="check" /></SelectItemIndicator>
					</SelectItem>
				</SelectViewport>
			</SelectContent>
		</SelectPortal>
	</SelectRoot>
</template>

<!-- Not scoped: the list is rendered in a portal at the end of <body>. -->
<style>
.m-select {
	display: inline-flex;
	align-items: center;
	justify-content: space-between;
	gap: var(--space-2);
	text-align: left;
	cursor: pointer;
}
.m-select-value {
	display: inline-flex;
	min-width: 0;
	overflow: hidden;
	align-items: center;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.m-select-chevron {
	display: inline-flex;
	color: var(--color-muted);
	transition: transform 120ms ease;
}
.m-select[data-state="open"] .m-select-chevron {
	transform: rotate(180deg);
}
.m-select-content {
	z-index: 50;
	min-width: var(--reka-select-trigger-width);
	max-height: min(22rem, var(--reka-select-content-available-height));
	overflow: hidden;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	box-shadow: var(--shadow-lg);
}
.m-select-viewport {
	padding: var(--space-1);
}
.m-select-item {
	position: relative;
	display: flex;
	min-height: 2.5rem;
	align-items: center;
	justify-content: space-between;
	gap: var(--space-3);
	padding: 0 var(--space-3);
	border-radius: var(--radius-sm);
	color: var(--color-ink);
	font-size: 0.9375rem;
	cursor: pointer;
	user-select: none;
}
/* The highlight shows the keyboard position; no outline on top of it. */
.m-select-item:focus-visible {
	outline: none;
}
.m-select-item[data-highlighted] {
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
}
.m-select-item[data-state="checked"] {
	font-weight: var(--font-weight-semibold);
}
.m-select-check {
	display: inline-flex;
	color: var(--color-accent-strong);
}
</style>
