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
		<div class="form-grid">
			<label
				>Command <InputText :model-value="modelValue.command" @update:model-value="updateCommand"
			/></label>
			<label
				>Working directory
				<InputText :model-value="modelValue.dir" @update:model-value="updateDirectory"
			/></label>
			<label
				>Timeout (seconds)
				<InputNumber
					:model-value="modelValue.timeoutSec"
					:min="1"
					@update:model-value="updateTimeout"
			/></label>
			<label
				>Output rule
				<Select
					:model-value="modelValue.rule.type"
					:options="['exit_code', 'match', 'not_match']"
					@update:model-value="updateRuleType"
			/></label>
			<label v-if="modelValue.rule.type !== 'exit_code'"
				>Regex pattern
				<InputText :model-value="modelValue.rule.pattern" @update:model-value="updatePattern"
			/></label>
		</div>
		<div class="dynamic-block">
			<strong>Arguments</strong>
			<InputText
				v-for="(argument, index) in args"
				:key="index"
				:model-value="argument"
				@update:model-value="updateArgument(index, $event)"
			/>
			<button type="button" @click="emit('add-argument')">+ Add argument</button>
		</div>
		<div class="dynamic-block">
			<strong>Environment</strong>
			<div v-for="(row, index) in environment" :key="index" class="env-row">
				<InputText
					placeholder="KEY"
					:model-value="row.key"
					@update:model-value="updateEnvironment(index, 'key', $event)"
				/>
				<InputText
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
	border-top: 1px solid #e5e1d8;
}
.action-editor h2 {
	margin: 0 0 1rem;
	font-size: 1.35rem;
	font-weight: 500;
}
.form-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(min(100%, 15rem), 1fr));
	gap: 1rem;
}
label {
	display: flex;
	flex-direction: column;
	gap: 0.4rem;
	color: #53615b;
	font-family: "Trebuchet MS", sans-serif;
	font-size: 0.85rem;
	font-weight: 700;
}
.dynamic-block {
	display: grid;
	gap: 0.5rem;
	margin-top: 1rem;
}
.dynamic-block button {
	width: fit-content;
	border: 0;
	background: transparent;
	color: #a24b32;
	cursor: pointer;
	font-family: "Trebuchet MS", sans-serif;
	font-weight: 700;
}
.env-row {
	display: grid;
	grid-template-columns: 1fr 2fr;
	gap: 0.5rem;
}
</style>
