<script setup lang="ts">
import { computed, ref, useId, watch } from "vue";
import Button from "primevue/button";
import InputNumber from "primevue/inputnumber";
import InputText from "primevue/inputtext";
import Textarea from "primevue/textarea";
import Select from "primevue/select";
import type { Action } from "@/api";
import { newRowId, type ActionFieldErrors, type EnvironmentRow } from "@/views/cardEditModel";
import { parseCommandLine } from "@/views/commandLine";

type RuleType = Action["rule"]["type"];

const props = defineProps<{
	title: string;
	errors?: ActionFieldErrors;
}>();

// The command and its arguments are edited as one line (ADR-0012) and shown
// split underneath; the parent splits the line again when it saves. The
// editor owns the environment rows: adding and removing happens here and the
// parent only receives the new array.
const action = defineModel<Action>({ required: true });
const commandLine = defineModel<string>("commandLine", { required: true });
const environment = defineModel<EnvironmentRow[]>("environment", { required: true });

const RULE_OPTIONS: { label: string; value: RuleType }[] = [
	{ label: "Exit code is zero", value: "exit_code" },
	{ label: "Output matches pattern", value: "match" },
	{ label: "Output does not match pattern", value: "not_match" },
];

const ids = useId();
const ruleLabelId = computed(() => `${ids}-rule`);
const commandHelpId = computed(() => `${ids}-command-help`);
const commandPreviewId = computed(() => `${ids}-command-preview`);

const parsed = computed(() => parseCommandLine(commandLine.value));
// A save error wins; otherwise a parse error shows while typing.
const commandError = computed(
	() => props.errors?.command || (parsed.value.ok ? "" : parsed.value.message),
);
// A stored argument may contain a line break (a two-line `sh -c` script
// written through the API). A text input would silently drop it, so such a
// line is edited in a text area that keeps it (ADR-0012). The area stays
// once shown, so removing the break does not swap the field and lose focus.
const multiLine = ref(false);
watch(
	commandLine,
	(line) => {
		if (/[\r\n]/.test(line)) multiLine.value = true;
	},
	{ immediate: true },
);
const preview = computed(() =>
	parsed.value.ok && parsed.value.command ? parsed.value : undefined,
);

function updateAction(patch: Partial<Action>) {
	action.value = { ...action.value, ...patch };
}

function updateCommandLine(value: unknown) {
	commandLine.value = String(value ?? "");
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
			<div class="col-12 field">
				<label :for="`${ids}-command`" class="field-label">Command line</label>
				<component
					:is="multiLine ? Textarea : InputText"
					:id="`${ids}-command`"
					class="command-line"
					:auto-resize="multiLine || undefined"
					:rows="multiLine ? 2 : undefined"
					:model-value="commandLine"
					placeholder="/usr/bin/ping -c 1 -W 2 192.168.1.10"
					spellcheck="false"
					autocapitalize="off"
					autocomplete="off"
					:invalid="Boolean(commandError)"
					:aria-invalid="Boolean(commandError)"
					:aria-describedby="`${commandHelpId} ${commandPreviewId}`"
					@update:model-value="updateCommandLine"
				/>
				<small v-if="commandError" class="field-error">{{ commandError }}</small>
				<small :id="commandHelpId" class="field-help">
					Separate arguments with spaces. Quote an argument that contains spaces, such as
					<code>'My Disk'</code>. No shell is used: pipes and variables are not available.
				</small>
				<div :id="commandPreviewId" class="command-preview" aria-live="polite">
					<template v-if="preview">
						<span class="preview-label">Runs</span>
						<ol class="preview-words" aria-label="Command and arguments">
							<li class="preview-command">{{ preview.command }}</li>
							<li v-for="(arg, index) in preview.args" :key="index" class="preview-arg">
								<span v-if="arg">{{ arg }}</span>
								<em v-else>empty</em>
							</li>
						</ol>
					</template>
				</div>
			</div>
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
.command-line {
	font-family: monospace;
}
.field-help {
	color: var(--color-muted);
	font-weight: normal;
	line-height: 1.5;
}
.command-preview {
	display: flex;
	flex-wrap: wrap;
	align-items: baseline;
	gap: var(--space-2);
	min-width: 0;
}
.preview-label {
	color: var(--color-muted);
}
.preview-words {
	display: flex;
	flex-wrap: wrap;
	gap: var(--space-1);
	min-width: 0;
	margin: 0;
	padding: 0;
	list-style: none;
}
.preview-words li {
	max-width: 100%;
	padding: 0 var(--space-2);
	overflow-wrap: anywhere;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	background: var(--color-canvas);
	color: var(--color-ink);
	font-family: monospace;
	white-space: pre-wrap;
}
.preview-words .preview-command {
	border-color: var(--color-accent);
	color: var(--color-accent);
}
.preview-words em {
	color: var(--color-muted);
	font-family: var(--font-ui);
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
