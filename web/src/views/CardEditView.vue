<script lang="ts">
import type { ActionCard as SavedCard } from "@/api";

// The card created by the last save. After creating a card the view moves to
// its edit route; the router may reuse or recreate this component, and either
// way the form shows the saved card with "Card saved" without a reload.
let justSaved: SavedCard | undefined;
</script>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { onBeforeRouteLeave, RouterLink, useRoute, useRouter } from "vue-router";
import Button from "primevue/button";
import Card from "primevue/card";
import InputNumber from "primevue/inputnumber";
import InputText from "primevue/inputtext";
import Message from "primevue/message";
import Select from "primevue/select";
import Textarea from "primevue/textarea";
import ToggleSwitch from "primevue/toggleswitch";
import { ApiError, createCard, getCard, updateCard, type ActionCard } from "@/api";
import { useConfirm } from "primevue/useconfirm";
import { CARD_ICON_OPTIONS } from "@/ui/icons";
import ActionEditor from "@/components/ActionEditor.vue";
import {
	actionFrom,
	argumentRows,
	copyAction,
	emptyAction,
	emptyCard,
	environmentRows,
	fieldErrorsFromServer,
	fingerprint,
	validate as validateCard,
	type ArgumentRow,
	type EnvironmentRow,
} from "@/views/cardEditModel";
import { singleParam } from "@/router/params";

const route = useRoute();
const router = useRouter();
const confirm = useConfirm();
const isNew = () => route.name === "card-new";
const cardID = () => singleParam(route.params.id);

const loading = ref(!isNew());
const saving = ref(false);
const error = ref("");
const notice = ref("");
const statusEnabled = ref(false);
const initialFingerprint = ref("");
const fieldErrors = reactive<Record<string, string>>({});

const form = reactive<ActionCard>(emptyCard());
const primaryArgs = ref<ArgumentRow[]>([]);
const statusArgs = ref<ArgumentRow[]>([]);
const primaryEnv = ref<EnvironmentRow[]>([]);
const statusEnv = ref<EnvironmentRow[]>([]);
// Bumped on every route change so a response for a previous card is ignored.
let loadGeneration = 0;

function formFingerprint() {
	return fingerprint({
		form,
		statusEnabled: statusEnabled.value,
		primaryArgs: primaryArgs.value,
		statusArgs: statusArgs.value,
		primaryEnv: primaryEnv.value,
		statusEnv: statusEnv.value,
	});
}

const isDirty = computed(
	() => initialFingerprint.value !== "" && initialFingerprint.value !== formFingerprint(),
);

function markClean() {
	initialFingerprint.value = formFingerprint();
}

function applyCard(card: ActionCard) {
	Object.assign(form, {
		...card,
		primary: copyAction(card.primary),
		status: card.status ? copyAction(card.status) : undefined,
	});
	statusEnabled.value = Boolean(card.status);
	primaryArgs.value = argumentRows(card.primary.args);
	statusArgs.value = argumentRows(card.status?.args);
	primaryEnv.value = environmentRows(card.primary.env);
	statusEnv.value = environmentRows(card.status?.env);
	markClean();
}

function actionErrors(scope: "primary" | "status") {
	return {
		command: fieldErrors[`${scope}Command`],
		dir: fieldErrors[`${scope}Dir`],
		timeoutSec: fieldErrors[`${scope}Timeout`],
		pattern: fieldErrors[`${scope}Pattern`],
		args: fieldErrors[`${scope}Args`],
		env: fieldErrors[`${scope}Env`],
	};
}

function validate() {
	for (const key of Object.keys(fieldErrors)) delete fieldErrors[key];
	const result = validateCard(form, statusEnabled.value);
	Object.assign(fieldErrors, result.fieldErrors);
	return result.firstError;
}

function applyServerErrors(saveError: ApiError) {
	const { fieldErrors: serverErrors, unmapped } = fieldErrorsFromServer(saveError.fields ?? {});
	Object.assign(fieldErrors, serverErrors);
	error.value = unmapped.length
		? `${saveError.message} (${unmapped.join("; ")})`
		: saveError.message;
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
		applyCard(saved);
		notice.value = "Card saved";
		// Stay in the edit context (UX spec §7.3); a new card moves to its own
		// edit route so a reload or a second save addresses the stored card.
		if (isNew()) {
			justSaved = saved;
			await router.replace(`/cards/${encodeURIComponent(saved.id)}/edit`);
		}
	} catch (saveError) {
		if (saveError instanceof ApiError && saveError.fields) applyServerErrors(saveError);
		else error.value = saveError instanceof Error ? saveError.message : "Unable to save card";
	} finally {
		saving.value = false;
	}
}

async function loadCard() {
	const generation = ++loadGeneration;
	error.value = "";
	notice.value = "";
	if (isNew()) {
		applyCard(emptyCard());
		loading.value = false;
		return;
	}
	if (justSaved?.id === cardID()) {
		applyCard(justSaved);
		justSaved = undefined;
		notice.value = "Card saved";
		loading.value = false;
		return;
	}
	loading.value = true;
	try {
		const loaded = await getCard(cardID());
		if (generation !== loadGeneration) return;
		applyCard(loaded);
	} catch (loadError) {
		if (generation !== loadGeneration) return;
		error.value = loadError instanceof Error ? loadError.message : "Unable to load card";
	} finally {
		if (generation === loadGeneration) loading.value = false;
	}
}

// The same component serves /cards/new/edit and /cards/:id/edit, so a route
// change reuses the instance and must reload the form instead of relying on
// mount.
watch(
	() => [route.name, singleParam(route.params.id)] as const,
	() => void loadCard(),
	{
		immediate: true,
	},
);

onMounted(() => {
	window.addEventListener("beforeunload", handleBeforeUnload);
});

function handleBeforeUnload(event: BeforeUnloadEvent) {
	if (!isDirty.value) return;
	event.preventDefault();
	event.returnValue = "";
}

function confirmDiscard(): Promise<boolean> {
	return new Promise((resolve) => {
		confirm.require({
			header: "Discard unsaved changes?",
			message: "Your edits to this card have not been saved.",
			icon: "pi pi-exclamation-triangle",
			acceptLabel: "Discard changes",
			rejectLabel: "Keep editing",
			acceptProps: { severity: "danger" },
			rejectProps: { severity: "secondary", outlined: true },
			defaultFocus: "reject",
			accept: () => resolve(true),
			reject: () => resolve(false),
			onHide: () => resolve(false),
		});
	});
}

onBeforeRouteLeave(async () => {
	if (!isDirty.value || saving.value) return true;
	return confirmDiscard();
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
							>Description
							<Textarea
								v-model="form.description"
								rows="2"
								:invalid="Boolean(fieldErrors.description)"
							/>
							<small v-if="fieldErrors.description" class="field-error">{{
								fieldErrors.description
							}}</small>
						</label>
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
					v-model:args="primaryArgs"
					v-model:environment="primaryEnv"
					title="Primary action"
					:errors="actionErrors('primary')"
				/>

				<div class="status-toggle">
					<ToggleSwitch v-model="statusEnabled" input-id="status-enabled" />
					<label for="status-enabled">Configure status action</label>
				</div>

				<ActionEditor
					v-if="statusEnabled"
					v-model:args="statusArgs"
					v-model:environment="statusEnv"
					:model-value="form.status ?? emptyAction()"
					title="Status action"
					:errors="actionErrors('status')"
					@update:model-value="form.status = $event"
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
.field-error {
	color: var(--color-danger);
	font-size: 0.78rem;
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
	position: sticky;
	bottom: 0;
	z-index: 1;
	margin: 0 calc(-1 * var(--space-4));
	padding: var(--space-3) var(--space-4);
	border-top: 1px solid var(--color-border);
	background: color-mix(in srgb, var(--color-surface) 94%, transparent);
	backdrop-filter: blur(4px);
}
.loading-state {
	display: grid;
	min-height: 320px;
	place-items: center;
	color: var(--color-muted);
}
</style>
