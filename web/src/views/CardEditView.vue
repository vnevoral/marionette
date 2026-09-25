<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { onBeforeRouteLeave, RouterLink, useRoute, useRouter } from "vue-router";
import Button from "primevue/button";
import Card from "primevue/card";
import InputNumber from "primevue/inputnumber";
import InputText from "primevue/inputtext";
import Message from "primevue/message";
import Select from "primevue/select";
import Textarea from "primevue/textarea";
import ToggleSwitch from "primevue/toggleswitch";
import {
	CARD_ICON_OPTIONS,
	createCard,
	getCard,
	updateCard,
	type Action,
	type ActionCard,
} from "@/api";
import ActionEditor from "@/components/ActionEditor.vue";

type EnvironmentRow = { key: string; value: string };

const route = useRoute();
const router = useRouter();
const isNew = () => route.name === "card-new";
const cardID = () => String(route.params.id);

const loading = ref(!isNew());
const saving = ref(false);
const error = ref("");
const notice = ref("");
const statusEnabled = ref(false);
const initialFingerprint = ref("");
const fieldErrors = reactive<Record<string, string>>({});

function emptyAction(): Action {
	return { command: "", args: [], dir: "", env: {}, timeoutSec: 30, rule: { type: "exit_code" } };
}

function emptyCard(): ActionCard {
	return {
		id: "",
		name: "",
		description: "",
		icon: CARD_ICON_OPTIONS[0].value,
		primary: emptyAction(),
	};
}

const form = reactive<ActionCard>(emptyCard());
const primaryArgs = ref<string[]>([]);
const statusArgs = ref<string[]>([]);
const primaryEnv = ref<EnvironmentRow[]>([]);
const statusEnv = ref<EnvironmentRow[]>([]);

function formFingerprint() {
	return JSON.stringify({
		name: form.name,
		description: form.description,
		icon: form.icon,
		primary: actionFrom(form.primary, primaryArgs.value, primaryEnv.value),
		status:
			statusEnabled.value && form.status
				? actionFrom(form.status, statusArgs.value, statusEnv.value)
				: undefined,
		pollingIntervalSeconds: form.pollingIntervalSeconds,
		fastPollingIntervalSeconds: form.fastPollingIntervalSeconds,
		fastPollingWindowSeconds: form.fastPollingWindowSeconds,
	});
}

const isDirty = computed(
	() => initialFingerprint.value !== "" && initialFingerprint.value !== formFingerprint(),
);

function markClean() {
	initialFingerprint.value = formFingerprint();
}

function copyAction(action: Action): Action {
	return {
		...action,
		args: [...(action.args ?? [])],
		env: { ...(action.env ?? {}) },
		rule: { ...action.rule },
	};
}

function applyCard(card: ActionCard) {
	Object.assign(form, {
		...card,
		primary: copyAction(card.primary),
		status: card.status ? copyAction(card.status) : undefined,
	});
	statusEnabled.value = Boolean(card.status);
	primaryArgs.value = [...(card.primary.args ?? [])];
	statusArgs.value = [...(card.status?.args ?? [])];
	primaryEnv.value = Object.entries(card.primary.env ?? {}).map(([key, value]) => ({ key, value }));
	statusEnv.value = Object.entries(card.status?.env ?? {}).map(([key, value]) => ({ key, value }));
	markClean();
}

function actionFrom(action: Action, args: string[], environment: EnvironmentRow[]): Action {
	const env = Object.fromEntries(
		environment.filter((row) => row.key.trim()).map((row) => [row.key.trim(), row.value]),
	);
	return {
		...action,
		args: args.map((arg) => arg.trim()).filter(Boolean),
		env,
		dir: action.dir?.trim(),
		rule: {
			...action.rule,
			pattern: action.rule.type === "exit_code" ? undefined : action.rule.pattern?.trim(),
		},
	};
}

function validate() {
	for (const key of Object.keys(fieldErrors)) delete fieldErrors[key];
	let firstError = "";
	const addError = (key: string, message: string) => {
		fieldErrors[key] = message;
		if (!firstError) firstError = message;
	};
	if (!form.name.trim()) addError("name", "Card name is required");
	if (!form.primary.command.trim()) addError("primaryCommand", "Primary command is required");
	if (form.primary.timeoutSec <= 0) addError("primaryTimeout", "Primary timeout must be positive");
	if (statusEnabled.value) {
		if (!form.status?.command.trim()) addError("statusCommand", "Status command is required");
		if (!form.status || form.status.timeoutSec <= 0)
			addError("statusTimeout", "Status timeout must be positive");
	}
	return firstError;
}

async function save() {
	error.value = validate();
	if (error.value) return;
	saving.value = true;
	notice.value = "";
	const payload: ActionCard = {
		...form,
		id: isNew() ? "" : cardID(),
		name: form.name.trim(),
		primary: actionFrom(form.primary, primaryArgs.value, primaryEnv.value),
		status:
			statusEnabled.value && form.status
				? actionFrom(form.status, statusArgs.value, statusEnv.value)
				: undefined,
	};
	try {
		const saved = isNew() ? await createCard(payload) : await updateCard(payload);
		markClean();
		notice.value = "Card saved";
		await router.push(`/cards/${encodeURIComponent(saved.id)}`);
	} catch (saveError) {
		error.value = saveError instanceof Error ? saveError.message : "Unable to save card";
	} finally {
		saving.value = false;
	}
}

function addArgument(target: "primary" | "status") {
	(target === "primary" ? primaryArgs : statusArgs).value.push("");
}

function addEnvironment(target: "primary" | "status") {
	(target === "primary" ? primaryEnv : statusEnv).value.push({ key: "", value: "" });
}

onMounted(async () => {
	if (isNew()) {
		loading.value = false;
		markClean();
		window.addEventListener("beforeunload", handleBeforeUnload);
		return;
	}
	try {
		applyCard(await getCard(cardID()));
	} catch (loadError) {
		error.value = loadError instanceof Error ? loadError.message : "Unable to load card";
	} finally {
		loading.value = false;
		window.addEventListener("beforeunload", handleBeforeUnload);
	}
});

function handleBeforeUnload(event: BeforeUnloadEvent) {
	if (!isDirty.value) return;
	event.preventDefault();
	event.returnValue = "";
}

onBeforeRouteLeave(() => {
	if (!isDirty.value || saving.value) return true;
	return window.confirm("You have unsaved changes. Leave without saving?");
});

onBeforeUnmount(() => {
	window.removeEventListener("beforeunload", handleBeforeUnload);
});
</script>

<template>
	<main class="edit-page">
		<header class="edit-header flex flex-column gap-3 mb-8">
			<RouterLink class="back-link" :to="isNew() ? '/' : `/cards/${encodeURIComponent(cardID())}`">
				<i class="pi pi-arrow-left" aria-hidden="true" />
				<span>{{ isNew() ? "Back to overview" : "Back to card detail" }}</span>
			</RouterLink>
			<div>
				<p class="eyebrow">{{ isNew() ? "NEW ACTION CARD" : "EDIT ACTION CARD" }}</p>
				<h1>{{ isNew() ? "New card" : "Edit card" }}</h1>
				<p class="page-lede">
					Configure the identity, actions, and automatic checks for this card.
				</p>
			</div>
		</header>

		<Message v-if="error" severity="error" :closable="false">{{ error }}</Message>
		<Message v-if="notice" severity="success" :closable="false">{{ notice }}</Message>

		<div v-if="loading" class="loading-state" role="status" aria-live="polite">
			<span>Loading card...</span>
		</div>

		<Card v-else class="editor-card">
			<template #content>
				<section aria-labelledby="identity-title">
					<div class="section-heading">
						<h2 id="identity-title">Card identity</h2>
						<p>Name the service and choose how it appears on the dashboard.</p>
					</div>
					<div class="form-grid grid">
						<label class="col-12 md:col-6"
							>Name
							<InputText v-model="form.name" :invalid="Boolean(fieldErrors.name)" />
							<small v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</small>
						</label>
						<label class="col-12"
							>Description <Textarea v-model="form.description" rows="2"
						/></label>
						<label class="col-12 md:col-6"
							>Icon
							<Select
								v-model="form.icon"
								:options="[...CARD_ICON_OPTIONS]"
								option-label="label"
								option-value="value"
							>
								<template #value="slotProps">
									<div v-if="slotProps.value" class="icon-option">
										<i :class="slotProps.value" aria-hidden="true" />
										<span>{{
											CARD_ICON_OPTIONS.find((icon) => icon.value === slotProps.value)?.label
										}}</span>
									</div>
								</template>
								<template #option="slotProps">
									<div class="icon-option">
										<i :class="slotProps.option.value" aria-hidden="true" />
										<span>{{ slotProps.option.label }}</span>
									</div>
								</template>
							</Select>
						</label>
					</div>
				</section>

				<ActionEditor
					v-model="form.primary"
					:args="primaryArgs"
					:environment="primaryEnv"
					title="Primary action"
					:errors="{ command: fieldErrors.primaryCommand, timeoutSec: fieldErrors.primaryTimeout }"
					@update:args="primaryArgs = $event"
					@update:environment="primaryEnv = $event"
					@add-argument="addArgument('primary')"
					@add-environment="addEnvironment('primary')"
				/>

				<div class="status-toggle">
					<ToggleSwitch v-model="statusEnabled" input-id="status-enabled" />
					<label for="status-enabled">Configure status action</label>
				</div>

				<ActionEditor
					v-if="statusEnabled"
					:model-value="form.status ?? emptyAction()"
					:args="statusArgs"
					:environment="statusEnv"
					title="Status action"
					:errors="{ command: fieldErrors.statusCommand, timeoutSec: fieldErrors.statusTimeout }"
					@update:model-value="form.status = $event"
					@update:args="statusArgs = $event"
					@update:environment="statusEnv = $event"
					@add-argument="addArgument('status')"
					@add-environment="addEnvironment('status')"
				/>

				<section class="polling-section" aria-labelledby="polling-title">
					<div class="section-heading">
						<h2 id="polling-title">Automatic checks</h2>
						<p>Keep the card status current without manual checks.</p>
					</div>
					<div class="form-grid polling-grid grid">
						<label class="col-12 md:col-4"
							>Polling interval (seconds)
							<InputNumber v-model="form.pollingIntervalSeconds" :min="0" />
						</label>
						<label class="col-12 md:col-4"
							>Fast interval (seconds)
							<InputNumber v-model="form.fastPollingIntervalSeconds" :min="0" />
						</label>
						<label class="col-12 md:col-4"
							>Fast window (seconds)
							<InputNumber v-model="form.fastPollingWindowSeconds" :min="0" />
						</label>
					</div>
				</section>
			</template>
			<template #footer>
				<div class="editor-actions flex flex-wrap justify-content-end gap-3">
					<RouterLink
						class="cancel-link"
						:to="isNew() ? '/' : `/cards/${encodeURIComponent(cardID())}`"
						>Cancel</RouterLink
					>
					<Button
						label="Save card"
						icon="pi pi-check"
						class="primary-action-button"
						:loading="saving"
						@click="save"
					/>
				</div>
			</template>
		</Card>
	</main>
</template>

<style scoped>
.edit-page {
	max-width: var(--page-max-width);
	margin: 0 auto;
	padding: var(--page-padding-y) var(--page-padding-x);
}
.edit-header {
	max-width: 760px;
}
.back-link,
.cancel-link {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	color: var(--color-accent-strong);
	font-weight: var(--font-weight-medium);
	text-decoration: none;
}
.back-link:hover,
.cancel-link:hover {
	color: var(--color-ink);
	text-decoration: underline;
}
.eyebrow {
	margin: 0 0 var(--space-3);
	color: var(--color-accent);
	font-size: 0.75rem;
	font-weight: var(--font-weight-semibold);
	letter-spacing: 0.14em;
}
h1,
h2 {
	margin: 0;
	font-family: var(--font-display);
	font-weight: 500;
}
h1 {
	font-size: clamp(2.5rem, 6vw, 4.5rem);
	line-height: 0.95;
}
h2 {
	font-size: 1.35rem;
}
.page-lede,
.section-heading p {
	margin: var(--space-3) 0 0;
	color: var(--color-muted);
}
.editor-card {
	max-width: 960px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	box-shadow: var(--shadow-subtle);
}
.editor-card :deep(.p-card-body) {
	background: var(--color-surface);
	color: var(--color-ink);
}
.section-heading {
	margin-bottom: var(--space-4);
}
.form-grid :deep(.p-inputtext),
.form-grid :deep(.p-textarea),
.form-grid :deep(.p-inputnumber),
.form-grid :deep(.p-select) {
	width: 100%;
}
.form-grid :deep(.p-inputtext),
.form-grid :deep(.p-textarea),
.form-grid :deep(.p-inputnumber-input),
.form-grid :deep(.p-select) {
	border-color: var(--color-border-strong);
	background: var(--color-surface-raised);
	color: var(--color-ink);
}
.form-grid :deep(.p-select-label) {
	color: var(--color-ink);
}
label {
	display: flex;
	flex-direction: column;
	gap: var(--space-2);
	color: var(--color-muted);
	font-size: 0.85rem;
	font-weight: var(--font-weight-medium);
}
.status-toggle {
	display: flex;
	align-items: center;
	gap: var(--space-3);
	margin-top: var(--space-8);
	color: var(--color-ink);
	font-weight: var(--font-weight-medium);
}
.polling-section {
	margin-top: var(--space-8);
	padding-top: var(--space-6);
	border-top: 1px solid var(--color-border);
}
.editor-actions {
	padding-top: var(--space-2);
}
.loading-state {
	display: grid;
	min-height: 320px;
	place-items: center;
	color: var(--color-muted);
}
</style>
