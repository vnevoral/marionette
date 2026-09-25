<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { RouterLink } from "vue-router";
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
	deleteCard,
	listCards,
	updateCard,
	type Action,
	type ActionCard,
} from "@/api";
import ActionEditor from "@/components/ActionEditor.vue";

type EnvironmentRow = { key: string; value: string };

const cards = ref<ActionCard[]>([]);
const selectedID = ref("");
const statusEnabled = ref(false);
const loading = ref(true);
const saving = ref(false);
const deleting = ref(false);
const error = ref("");
const notice = ref("");

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

function copyAction(action: Action): Action {
	return {
		...action,
		args: [...(action.args ?? [])],
		env: { ...(action.env ?? {}) },
		rule: { ...action.rule },
	};
}

function selectCard(cardID: string) {
	const card = cards.value.find((item) => item.id === cardID);
	if (!card) return;
	Object.assign(form, {
		...card,
		primary: copyAction(card.primary),
		status: card.status ? copyAction(card.status) : undefined,
	});
	selectedID.value = cardID;
	statusEnabled.value = Boolean(card.status);
	primaryArgs.value = [...(card.primary.args ?? [])];
	statusArgs.value = [...(card.status?.args ?? [])];
	primaryEnv.value = Object.entries(card.primary.env ?? {}).map(([key, value]) => ({ key, value }));
	statusEnv.value = Object.entries(card.status?.env ?? {}).map(([key, value]) => ({ key, value }));
	error.value = "";
	notice.value = "";
}

function newCard() {
	Object.assign(form, emptyCard());
	selectedID.value = "";
	statusEnabled.value = false;
	primaryArgs.value = [];
	statusArgs.value = [];
	primaryEnv.value = [];
	statusEnv.value = [];
	error.value = "";
	notice.value = "";
}

async function loadCards() {
	loading.value = true;
	try {
		cards.value = await listCards();
		if (selectedID.value) selectCard(selectedID.value);
	} catch (loadError) {
		error.value = loadError instanceof Error ? loadError.message : "Unable to load cards";
	} finally {
		loading.value = false;
	}
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
	if (!form.name.trim()) return "Card name is required";
	if (!form.primary.command.trim()) return "Primary command is required";
	if (form.primary.timeoutSec <= 0) return "Primary timeout must be positive";
	if (statusEnabled.value) {
		if (!form.status?.command.trim()) return "Status command is required";
		if (!form.status || form.status.timeoutSec <= 0) return "Status timeout must be positive";
	}
	return "";
}

async function save() {
	error.value = validate();
	if (error.value) return;
	saving.value = true;
	notice.value = "";
	const payload: ActionCard = {
		...form,
		id: form.id.trim(),
		name: form.name.trim(),
		primary: actionFrom(form.primary, primaryArgs.value, primaryEnv.value),
		status:
			statusEnabled.value && form.status
				? actionFrom(form.status, statusArgs.value, statusEnv.value)
				: undefined,
	};
	try {
		const saved = selectedID.value ? await updateCard(payload) : await createCard(payload);
		cards.value = await listCards();
		selectCard(saved.id);
		notice.value = "Card saved";
	} catch (saveError) {
		error.value = saveError instanceof Error ? saveError.message : "Unable to save card";
	} finally {
		saving.value = false;
	}
}

async function removeCard(cardID: string, cardName: string) {
	if (!window.confirm(`Delete ${cardName}?`)) return;
	deleting.value = true;
	error.value = "";
	try {
		await deleteCard(cardID);
		cards.value = await listCards();
		if (selectedID.value === cardID) newCard();
		notice.value = "Card deleted";
	} catch (deleteError) {
		error.value = deleteError instanceof Error ? deleteError.message : "Unable to delete card";
	} finally {
		deleting.value = false;
	}
}

function addArgument(target: "primary" | "status") {
	(target === "primary" ? primaryArgs : statusArgs).value.push("");
}

function addEnvironment(target: "primary" | "status") {
	(target === "primary" ? primaryEnv : statusEnv).value.push({ key: "", value: "" });
}

onMounted(async () => {
	await loadCards();
	newCard();
});
</script>

<template>
	<main class="manage-shell">
		<header
			class="manage-header flex flex-column md:flex-row align-items-start md:align-items-end justify-content-between gap-6 mb-6"
		>
			<div>
				<p class="eyebrow">MARIONETTE CONTROL</p>
				<h1>Manage cards</h1>
			</div>
			<RouterLink class="back-link" to="/">Back to dashboard</RouterLink>
		</header>
		<Message v-if="error" severity="error" :closable="false">{{ error }}</Message>
		<Message v-if="notice" severity="success" :closable="false">{{ notice }}</Message>
		<div class="manage-layout grid">
			<Card class="card-list col-12 lg:col-4">
				<template #title>Cards</template>
				<template #content>
					<Button
						v-if="cards.length"
						label="New card"
						icon="pi pi-plus"
						class="new-button"
						@click="newCard"
					/>
					<div v-if="loading" class="muted">Loading...</div>
					<div
						v-else-if="!cards.length"
						class="empty-list flex flex-column align-items-start gap-3 p-4"
					>
						<p class="muted">No cards configured.</p>
						<Button label="New card" icon="pi pi-plus" @click="newCard" />
					</div>
					<article
						v-for="card in cards"
						:key="card.id"
						class="card-row"
						:class="{ selected: selectedID === card.id }"
					>
						<div class="card-row-info flex flex-column gap-1">
							<strong>{{ card.name }}</strong>
							<span>{{ card.status ? "Status check enabled" : "No status check" }}</span>
						</div>
						<div class="card-row-actions flex align-items-center gap-1">
							<Button
								label="Edit"
								icon="pi pi-pencil"
								size="small"
								outlined
								@click="selectCard(card.id)"
							/>
							<Button
								label="Delete"
								icon="pi pi-trash"
								size="small"
								severity="danger"
								text
								:loading="deleting"
								@click="removeCard(card.id, card.name)"
							/>
						</div>
					</article>
				</template>
			</Card>
			<Card class="editor-card col-12 lg:col-8">
				<template #title>{{ selectedID ? "Edit card" : "New card" }}</template>
				<template #content>
					<div class="form-grid grid">
						<label class="col-12 md:col-6">Name <InputText v-model="form.name" /></label>
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
								class="icon-select"
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
					<ActionEditor
						v-model="form.primary"
						:args="primaryArgs"
						:environment="primaryEnv"
						title="Primary action"
						@update:args="primaryArgs = $event"
						@update:environment="primaryEnv = $event"
						@add-argument="addArgument('primary')"
						@add-environment="addEnvironment('primary')"
					/>
					<div class="status-toggle">
						<ToggleSwitch v-model="statusEnabled" input-id="status-enabled" /><label
							for="status-enabled"
							>Configure status action</label
						>
					</div>
					<ActionEditor
						v-if="statusEnabled"
						:model-value="form.status ?? emptyAction()"
						:args="statusArgs"
						:environment="statusEnv"
						title="Status action"
						@update:model-value="form.status = $event"
						@update:args="statusArgs = $event"
						@update:environment="statusEnv = $event"
						@add-argument="addArgument('status')"
						@add-environment="addEnvironment('status')"
					/>
					<div class="form-grid polling-grid grid">
						<label class="col-12 md:col-4"
							>Polling interval (seconds)
							<InputNumber v-model="form.pollingIntervalSeconds" :min="0"
						/></label>
						<label class="col-12 md:col-4"
							>Fast interval (seconds)
							<InputNumber v-model="form.fastPollingIntervalSeconds" :min="0"
						/></label>
						<label class="col-12 md:col-4"
							>Fast window (seconds) <InputNumber v-model="form.fastPollingWindowSeconds" :min="0"
						/></label>
					</div>
				</template>
				<template #footer>
					<div class="editor-actions flex flex-wrap gap-3">
						<Button label="Save card" icon="pi pi-check" :loading="saving" @click="save" />
					</div>
				</template>
			</Card>
		</div>
	</main>
</template>

<style scoped>
:global(body) {
	margin: 0;
	background: var(--color-canvas);
	color: var(--color-ink);
	font-family: var(--font-ui);
}
.manage-shell {
	padding: clamp(2rem, 6vw, 5rem) clamp(1rem, 5vw, 5rem);
	background: var(--color-canvas);
}
.eyebrow {
	margin: 0 0 0.75rem;
	color: var(--color-accent);
	font-family: var(--font-ui);
	font-size: 0.75rem;
	font-weight: 700;
	letter-spacing: 0.14em;
}
h1,
h2 {
	margin: 0;
	font-weight: 500;
}
h1 {
	font-size: clamp(2.5rem, 6vw, 5rem);
	line-height: 0.95;
}
.back-link,
.manage-link {
	color: var(--color-accent);
	font-family: var(--font-ui);
	font-weight: 700;
	text-decoration: none;
}
.manage-link {
	align-self: center;
	margin-left: auto;
}
.card-list,
.editor-card {
	border: 1px solid var(--color-border);
	box-shadow: var(--shadow-subtle);
}
.new-button {
	width: 100%;
	margin-bottom: 1rem;
}
.muted {
	color: var(--color-muted);
	font-family: var(--font-ui);
}
.card-row {
	padding: 0.85rem 0;
	border-top: 1px solid var(--color-border);
	background: transparent;
	color: var(--color-ink);
	text-align: left;
}
.card-row.selected {
	margin: 0 -0.75rem;
	padding-right: 0.75rem;
	padding-left: 0.75rem;
	background: var(--color-accent-soft);
}
.card-row-info {
	min-width: 0;
}
.card-row-info strong {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.card-row-info small,
.card-row-info span {
	color: var(--color-muted);
	font-size: 0.75rem;
}
.empty-list p {
	margin: 0;
}
label {
	display: flex;
	flex-direction: column;
	gap: 0.4rem;
	color: var(--color-muted);
	font-family: var(--font-ui);
	font-size: 0.85rem;
	font-weight: 700;
}
.action-editor {
	margin-top: 2rem;
	padding-top: 1.5rem;
	border-top: 1px solid var(--color-border);
}
.action-editor h2 {
	margin-bottom: 1rem;
	font-size: 1.35rem;
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
	color: var(--color-accent);
	cursor: pointer;
	font-family: var(--font-ui);
	font-weight: 700;
}
.env-row {
	display: grid;
	grid-template-columns: 1fr 2fr;
	gap: 0.5rem;
}
.status-toggle {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	margin-top: 2rem;
	font-family: var(--font-ui);
}
.polling-grid {
	margin-top: 2rem;
}
@media (max-width: 48rem) {
	.manage-link {
		margin-left: 0;
	}
}
</style>
