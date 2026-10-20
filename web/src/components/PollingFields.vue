<script setup lang="ts">
import FormSection from "@/components/FormSection.vue";
import MNumberInput from "@/ui/MNumberInput.vue";

defineProps<{
	errors: {
		pollingIntervalSeconds?: string;
		fastPollingIntervalSeconds?: string;
		fastPollingWindowSeconds?: string;
	};
}>();

const interval = defineModel<number | undefined>("pollingIntervalSeconds", { required: true });
const fastInterval = defineModel<number | undefined>("fastPollingIntervalSeconds", {
	required: true,
});
const fastWindow = defineModel<number | undefined>("fastPollingWindowSeconds", { required: true });
</script>

<template>
	<FormSection
		title="Automatic checks"
		description="Keep the card status current without manual checks."
		heading-id="polling-title"
	>
		<div class="form-grid grid">
			<label class="col-12 md:col-4"
				>Polling interval (seconds)
				<MNumberInput
					v-model="interval"
					:min="0"
					:invalid="Boolean(errors.pollingIntervalSeconds)"
				/>
				<small v-if="errors.pollingIntervalSeconds" class="field-error">{{
					errors.pollingIntervalSeconds
				}}</small>
			</label>
			<label class="col-12 md:col-4"
				>Fast interval (seconds)
				<MNumberInput
					v-model="fastInterval"
					:min="0"
					:invalid="Boolean(errors.fastPollingIntervalSeconds)"
				/>
				<small v-if="errors.fastPollingIntervalSeconds" class="field-error">{{
					errors.fastPollingIntervalSeconds
				}}</small>
			</label>
			<label class="col-12 md:col-4"
				>Fast window (seconds)
				<MNumberInput
					v-model="fastWindow"
					:min="0"
					:invalid="Boolean(errors.fastPollingWindowSeconds)"
				/>
				<small v-if="errors.fastPollingWindowSeconds" class="field-error">{{
					errors.fastPollingWindowSeconds
				}}</small>
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
