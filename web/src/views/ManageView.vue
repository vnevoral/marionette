<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { RouterLink, useRouter } from "vue-router";
import Button from "primevue/button";
import Card from "primevue/card";
import InputNumber from "primevue/inputnumber";
import InputText from "primevue/inputtext";
import Message from "primevue/message";
import Textarea from "primevue/textarea";
import ToggleSwitch from "primevue/toggleswitch";
import { createCard, deleteCard, listCards, updateCard, type Action, type ActionCard } from "@/api";
import ActionEditor from "@/components/ActionEditor.vue";

type EnvironmentRow = { key: string; value: string };

const router = useRouter();
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
	return { id: "", name: "", description: "", icon: "", primary: emptyAction() };
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

async function remove() {
	if (!selectedID.value || !window.confirm(`Delete ${form.name}?`)) return;
	deleting.value = true;
	error.value = "";
	try {
		await deleteCard(selectedID.value);
		await router.push("/");
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
		<header class="manage-header">
			<div>
				<p class="eyebrow">MARIONETTE CONTROL</p>
				<h1>Manage cards</h1>
			</div>
			<RouterLink class="back-link" to="/">Back to dashboard</RouterLink>
		</header>
		<Message v-if="error" severity="error" :closable="false">{{ error }}</Message>
		<Message v-if="notice" severity="success" :closable="false">{{ notice }}</Message>
		<div class="manage-layout">
			<Card class="card-list">
				<template #title>Cards</template>
				<template #content>
					<Button label="New card" icon="pi pi-plus" class="new-button" @click="newCard" />
					<div v-if="loading" class="muted">Loading...</div>
					<div v-else-if="!cards.length" class="muted">No cards configured.</div>
					<button
						v-for="card in cards"
						:key="card.id"
						class="card-choice"
						:class="{ selected: selectedID === card.id }"
						@click="selectCard(card.id)"
					>
						<strong>{{ card.name }}</strong
						><small>{{ card.id }}</small>
					</button>
				</template>
			</Card>
			<Card class="editor-card">
				<template #title>{{ selectedID ? "Edit card" : "New card" }}</template>
				<template #content>
					<div class="form-grid">
						<label>ID <InputText v-model="form.id" :disabled="Boolean(selectedID)" /></label>
						<label>Name <InputText v-model="form.name" /></label>
						<label>Description <Textarea v-model="form.description" rows="2" /></label>
						<label>Icon <InputText v-model="form.icon" placeholder="pi pi-desktop" /></label>
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
					<div class="form-grid polling-grid">
						<label
							>Polling interval (seconds)
							<InputNumber v-model="form.pollingIntervalSeconds" :min="0"
						/></label>
						<label
							>Fast interval (seconds)
							<InputNumber v-model="form.fastPollingIntervalSeconds" :min="0"
						/></label>
						<label
							>Fast window (seconds) <InputNumber v-model="form.fastPollingWindowSeconds" :min="0"
						/></label>
					</div>
				</template>
				<template #footer>
					<div class="editor-actions">
						<Button label="Save" icon="pi pi-check" :loading="saving" @click="save" /><Button
							v-if="selectedID"
							label="Delete"
							icon="pi pi-trash"
							severity="danger"
							outlined
							:loading="deleting"
							@click="remove"
						/>
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
	min-height: 100vh;
	padding: clamp(2rem, 6vw, 5rem) clamp(1rem, 5vw, 5rem);
	background: var(--color-canvas);
}
.manage-header {
	display: flex;
	align-items: end;
	justify-content: space-between;
	max-width: 78rem;
	margin: 0 auto 2rem;
	gap: 2rem;
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
.manage-layout {
	display: grid;
	grid-template-columns: minmax(15rem, 20rem) minmax(0, 1fr);
	gap: 1.25rem;
	max-width: 78rem;
	margin: 0 auto;
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
.card-choice {
	display: flex;
	width: 100%;
	flex-direction: column;
	align-items: flex-start;
	gap: 0.25rem;
	padding: 0.75rem;
	border: 0;
	border-top: 1px solid var(--color-border);
	background: transparent;
	color: var(--color-ink);
	cursor: pointer;
	text-align: left;
}
.card-choice.selected {
	background: var(--color-accent-soft);
}
.card-choice small {
	color: var(--color-muted);
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
.editor-actions {
	display: flex;
	flex-wrap: wrap;
	gap: 0.75rem;
}
@media (max-width: 48rem) {
	.manage-header,
	.manage-layout {
		grid-template-columns: 1fr;
		flex-direction: column;
		align-items: stretch;
	}
	.manage-link {
		margin-left: 0;
	}
}
</style>
