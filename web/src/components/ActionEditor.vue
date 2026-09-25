<script setup lang="ts">
import InputNumber from "primevue/inputnumber";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import type { Action } from "@/api";

type EnvironmentRow = { key: string; value: string };
type RuleType = Action["rule"]["type"];

const props = defineProps<{
	modelValue: Action;
	args: string[];
	environment: EnvironmentRow[];
	title: string;
	errors?: {
		command?: string;
		timeoutSec?: string;
	};
}>();

const emit = defineEmits<{
	(e: "update:modelValue", value: Action): void;
	(e: "update:args", value: string[]): void;
	(e: "update:environment", value: EnvironmentRow[]): void;
	(e: "add-argument"): void;
	(e: "add-environment"): void;
}>();

function updateAction(key: keyof Action, value: unknown) {
	emit("update:modelValue", { ...props.modelValue, [key]: value });
}

function updateCommand(value: unknown) {
	updateAction("command", String(value ?? ""));
}

function updateDirectory(value: unknown) {
	updateAction("dir", String(value ?? ""));
}

function updateTimeout(value: unknown) {
	updateAction("timeoutSec", Number(value ?? 0));
}

function updateRuleType(value: unknown) {
	updateAction("rule", { ...props.modelValue.rule, type: value as RuleType });
}

function updatePattern(value: unknown) {
	updateAction("rule", { ...props.modelValue.rule, pattern: String(value ?? "") });
}

function updateArgument(index: number, value: unknown) {
	const args = [...props.args];
	args[index] = String(value ?? "");
	emit("update:args", args);
}

function updateEnvironment(index: number, key: "key" | "value", value: unknown) {
	const environment = props.environment.map((row) => ({ ...row }));
	environment[index][key] = String(value ?? "");
	emit("update:environment", environment);
}
</script>

<template>
	<section class="action-editor">
		<h2>{{ title }}</h2>
		<div class="form-grid grid">
			<label class="col-12 md:col-6"
				>Command
				<InputText
					:model-value="modelValue.command"
					:invalid="Boolean(props.errors?.command)"
					:aria-invalid="Boolean(props.errors?.command)"
					@update:model-value="updateCommand"
				/>
				<small v-if="props.errors?.command" class="field-error">{{ props.errors.command }}</small>
			</label>
			<label class="col-12 md:col-6"
				>Working directory
				<InputText :model-value="modelValue.dir" @update:model-value="updateDirectory"
			/></label>
			<label class="col-12 md:col-6 lg:col-4"
				>Timeout (seconds)
				<InputNumber
					:model-value="modelValue.timeoutSec"
					:min="1"
					:invalid="Boolean(props.errors?.timeoutSec)"
					:aria-invalid="Boolean(props.errors?.timeoutSec)"
					@update:model-value="updateTimeout"
				/>
				<small v-if="props.errors?.timeoutSec" class="field-error">{{
					props.errors.timeoutSec
				}}</small>
			</label>
			<label class="col-12 md:col-6 lg:col-4"
				>Output rule
				<Select
					:model-value="modelValue.rule.type"
					:options="['exit_code', 'match', 'not_match']"
					@update:model-value="updateRuleType"
			/></label>
			<label v-if="modelValue.rule.type !== 'exit_code'" class="col-12 md:col-6 lg:col-4"
				>Regex pattern
				<InputText :model-value="modelValue.rule.pattern" @update:model-value="updatePattern"
			/></label>
		</div>
		<div class="dynamic-block flex flex-column gap-2 mt-4">
			<strong>Arguments</strong>
			<InputText
				v-for="(argument, index) in args"
				:key="index"
				:model-value="argument"
				@update:model-value="updateArgument(index, $event)"
			/>
			<button type="button" @click="emit('add-argument')">+ Add argument</button>
		</div>
		<div class="dynamic-block flex flex-column gap-2 mt-4">
			<strong>Environment</strong>
			<div v-for="(row, index) in environment" :key="index" class="env-row grid">
				<InputText
					class="col-12 md:col-4"
					placeholder="KEY"
					:model-value="row.key"
					@update:model-value="updateEnvironment(index, 'key', $event)"
				/>
				<InputText
					class="col-12 md:col-8"
					placeholder="value"
					:model-value="row.value"
					@update:model-value="updateEnvironment(index, 'value', $event)"
				/>
			</div>
			<button type="button" @click="emit('add-environment')">+ Add variable</button>
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
label {
	display: flex;
	flex-direction: column;
	gap: 0.4rem;
	color: var(--color-muted);
	font-family: var(--font-ui);
	font-size: 0.85rem;
	font-weight: var(--font-weight-medium);
}
.field-error {
	color: var(--color-danger);
	font-size: 0.78rem;
}
.dynamic-block button {
	width: fit-content;
	border: 0;
	background: transparent;
	color: var(--color-accent);
	cursor: pointer;
	font-family: var(--font-ui);
	font-weight: var(--font-weight-medium);
}
</style>
