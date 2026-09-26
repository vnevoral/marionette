<script setup lang="ts">
import { computed, useId } from "vue";
import Button from "primevue/button";
import InputNumber from "primevue/inputnumber";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import type { Action } from "@/api";
import {
	newRowId,
	type ActionFieldErrors,
	type ArgumentRow,
	type EnvironmentRow,
} from "@/views/cardEditModel";

type RuleType = Action["rule"]["type"];

const props = defineProps<{
	title: string;
	errors?: ActionFieldErrors;
}>();

// The editor owns the repeatable rows: adding and removing happens here and
// the parent only receives the new arrays.
const action = defineModel<Action>({ required: true });
const args = defineModel<ArgumentRow[]>("args", { required: true });
const environment = defineModel<EnvironmentRow[]>("environment", { required: true });

const RULE_OPTIONS: { label: string; value: RuleType }[] = [
	{ label: "Exit code is zero", value: "exit_code" },
	{ label: "Output matches pattern", value: "match" },
	{ label: "Output does not match pattern", value: "not_match" },
];

const ids = useId();
const ruleLabelId = computed(() => `${ids}-rule`);

function updateAction(patch: Partial<Action>) {
	action.value = { ...action.value, ...patch };
}

function updateCommand(value: unknown) {
	updateAction({ command: String(value ?? "") });
}

function updateDirectory(value: unknown) {
	updateAction({ dir: String(value ?? "") });
}

function updateTimeout(value: unknown) {
	updateAction({ timeoutSec: Number(value ?? 0) });
}

function updateRuleType(value: unknown) {
	updateAction({ rule: { ...action.value.rule, type: value as RuleType } });
}

function updatePattern(value: unknown) {
	updateAction({ rule: { ...action.value.rule, pattern: String(value ?? "") } });
}

function updateArgument(id: string, value: unknown) {
	args.value = args.value.map((row) =>
		row.id === id ? { ...row, value: String(value ?? "") } : row,
	);
}

function addArgument() {
	args.value = [...args.value, { id: newRowId(), value: "" }];
}

function removeArgument(id: string) {
	args.value = args.value.filter((row) => row.id !== id);
}

function updateEnvironment(id: string, key: "key" | "value", value: unknown) {
	environment.value = environment.value.map((row) =>
		row.id === id ? { ...row, [key]: String(value ?? "") } : row,
	);
}

function addEnvironment() {
	environment.value = [...environment.value, { id: newRowId(), key: "", value: "" }];
}

function removeEnvironment(id: string) {
	environment.value = environment.value.filter((row) => row.id !== id);
}
</script>

<template>
	<section class="action-editor">
		<h2>{{ title }}</h2>
		<div class="form-grid grid">
			<label class="col-12 md:col-6"
				>Command
				<InputText
					:model-value="action.command"
					:invalid="Boolean(props.errors?.command)"
					:aria-invalid="Boolean(props.errors?.command)"
					@update:model-value="updateCommand"
				/>
				<small v-if="props.errors?.command" class="field-error">{{ props.errors.command }}</small>
			</label>
			<label class="col-12 md:col-6"
				>Working directory
				<InputText
					:model-value="action.dir"
					:invalid="Boolean(props.errors?.dir)"
					@update:model-value="updateDirectory"
				/>
				<small v-if="props.errors?.dir" class="field-error">{{ props.errors.dir }}</small>
			</label>
			<label class="col-12 md:col-6 lg:col-4"
				>Timeout (seconds)
				<InputNumber
					:model-value="action.timeoutSec"
					:min="1"
					:invalid="Boolean(props.errors?.timeoutSec)"
					:aria-invalid="Boolean(props.errors?.timeoutSec)"
					@update:model-value="updateTimeout"
				/>
				<small v-if="props.errors?.timeoutSec" class="field-error">{{
					props.errors.timeoutSec
				}}</small>
			</label>
			<div class="col-12 md:col-6 lg:col-4 field">
				<span :id="ruleLabelId" class="field-label">Output rule</span>
				<Select
					:model-value="action.rule.type"
					:options="RULE_OPTIONS"
					option-label="label"
					option-value="value"
					:aria-labelledby="ruleLabelId"
					@update:model-value="updateRuleType"
				/>
			</div>
			<label v-if="action.rule.type !== 'exit_code'" class="col-12 md:col-6 lg:col-4"
				>Regex pattern
				<InputText
					:model-value="action.rule.pattern"
					:invalid="Boolean(props.errors?.pattern)"
					@update:model-value="updatePattern"
				/>
				<small v-if="props.errors?.pattern" class="field-error">{{ props.errors.pattern }}</small>
			</label>
		</div>
		<div class="dynamic-block flex flex-column gap-2 mt-4">
			<strong>Arguments</strong>
			<small v-if="props.errors?.args" class="field-error">{{ props.errors.args }}</small>
			<div v-for="(row, index) in args" :key="row.id" class="row-line">
				<InputText
					:model-value="row.value"
					:aria-label="`Argument ${index + 1}`"
					@update:model-value="updateArgument(row.id, $event)"
				/>
				<Button
					type="button"
					icon="pi pi-times"
					text
					rounded
					severity="secondary"
					:aria-label="`Remove argument ${index + 1}`"
					@click="removeArgument(row.id)"
				/>
			</div>
			<Button
				type="button"
				label="Add argument"
				icon="pi pi-plus"
				text
				size="small"
				class="add-row"
				@click="addArgument"
			/>
		</div>
		<div class="dynamic-block flex flex-column gap-2 mt-4">
			<strong>Environment</strong>
			<small v-if="props.errors?.env" class="field-error">{{ props.errors.env }}</small>
			<div v-for="(row, index) in environment" :key="row.id" class="row-line env-row">
				<InputText
					class="env-key"
					placeholder="KEY"
					:model-value="row.key"
					:aria-label="`Variable ${index + 1} name`"
					@update:model-value="updateEnvironment(row.id, 'key', $event)"
				/>
				<InputText
					class="env-value"
					placeholder="value"
					:model-value="row.value"
					:aria-label="`Variable ${index + 1} value`"
					@update:model-value="updateEnvironment(row.id, 'value', $event)"
				/>
				<Button
					type="button"
					icon="pi pi-times"
					text
					rounded
					severity="secondary"
					:aria-label="`Remove variable ${index + 1}`"
					@click="removeEnvironment(row.id)"
				/>
			</div>
			<Button
				type="button"
				label="Add variable"
				icon="pi pi-plus"
				text
				size="small"
				class="add-row"
				@click="addEnvironment"
			/>
		</div>
	</section>
</template>

<style scoped>
.action-editor {
	margin-top: 2rem;
	padding-top: 1.5rem;
	border-top: 1px solid var(--color-border);
}
.action-editor h2 {
	margin: 0 0 1rem;
	font-size: 1.35rem;
	font-weight: 500;
}
label,
.field {
	display: flex;
	flex-direction: column;
	gap: 0.4rem;
	color: var(--color-muted);
	font-family: var(--font-ui);
	font-size: 0.85rem;
	font-weight: var(--font-weight-medium);
}
.row-line {
	display: flex;
	align-items: center;
	gap: var(--space-2);
}
.row-line :deep(.p-inputtext) {
	flex: 1 1 auto;
	min-width: 0;
}
.env-row .env-key {
	flex: 1 1 30%;
}
.env-row .env-value {
	flex: 1 1 60%;
}
.add-row {
	align-self: flex-start;
}
@media (max-width: 30rem) {
	.env-row {
		flex-wrap: wrap;
	}
	.env-row .env-key,
	.env-row .env-value {
		flex-basis: 100%;
	}
}
</style>
