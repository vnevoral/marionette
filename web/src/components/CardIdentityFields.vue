<script setup lang="ts">
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import Textarea from "primevue/textarea";
import FormSection from "@/components/FormSection.vue";
import { CARD_ICON_OPTIONS } from "@/ui/icons";

defineProps<{ errors: { name?: string; description?: string; icon?: string } }>();

const name = defineModel<string>("name", { required: true });
const description = defineModel<string | undefined>("description", { required: true });
const icon = defineModel<string | undefined>("icon", { required: true });

function iconLabel(value: unknown) {
	return CARD_ICON_OPTIONS.find((option) => option.value === value)?.label;
}
</script>

<template>
	<FormSection
		title="Card identity"
		description="Name the service and choose how it appears on the dashboard."
		heading-id="identity-title"
	>
		<div class="form-grid grid">
			<label class="col-12 md:col-6"
				>Name
				<InputText v-model="name" :invalid="Boolean(errors.name)" />
				<small v-if="errors.name" class="field-error">{{ errors.name }}</small>
			</label>
			<label class="col-12"
				>Description
				<Textarea v-model="description" rows="2" :invalid="Boolean(errors.description)" />
				<small v-if="errors.description" class="field-error">{{ errors.description }}</small>
			</label>
			<label class="col-12 md:col-6"
				>Icon
				<Select
					v-model="icon"
					:options="[...CARD_ICON_OPTIONS]"
					option-label="label"
					option-value="value"
					:invalid="Boolean(errors.icon)"
					aria-label="Icon"
				>
					<template #value="slotProps">
						<div v-if="slotProps.value" class="icon-option">
							<i :class="slotProps.value" aria-hidden="true" />
							<span>{{ iconLabel(slotProps.value) }}</span>
						</div>
					</template>
					<template #option="slotProps">
						<div class="icon-option">
							<i :class="slotProps.option.value" aria-hidden="true" />
							<span>{{ slotProps.option.label }}</span>
						</div>
					</template>
				</Select>
				<small v-if="errors.icon" class="field-error">{{ errors.icon }}</small>
			</label>
		</div>
	</FormSection>
</template>

<style scoped>
label {
	display: flex;
	flex-direction: column;
	gap: var(--space-2);
	color: var(--color-muted);
	font-size: 0.85rem;
	font-weight: var(--font-weight-medium);
}
</style>
