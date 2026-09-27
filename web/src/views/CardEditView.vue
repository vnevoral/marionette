<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import Card from "primevue/card";
import ToggleSwitch from "primevue/toggleswitch";
import { ApiError, createCard, getCard, updateCard, type ActionCard } from "@/api";
import ActionEditor from "@/components/ActionEditor.vue";
import CardIdentityFields from "@/components/CardIdentityFields.vue";
import PageHeader from "@/components/PageHeader.vue";
import PollingFields from "@/components/PollingFields.vue";
import RequestState from "@/components/RequestState.vue";
import SaveBar from "@/components/SaveBar.vue";
import { useNotify } from "@/composables/useNotify";
import { useUnsavedChangesGuard } from "@/composables/useUnsavedChangesGuard";
import { knownCardColor } from "@/ui/cardColors";
import { ACTIONS, FEEDBACK, LOADING } from "@/ui/vocabulary";
import {
	actionFrom,
	commandLineOf,
	copyAction,
	emptyAction,
	emptyCard,
	environmentRows,
	fieldErrorsFromServer,
	fingerprint,
	rememberSavedCard,
	takeSavedCard,
	validate as validateCard,
	type EnvironmentRow,
} from "@/views/cardEditModel";
import { singleParam } from "@/router/params";

const route = useRoute();
const notify = useNotify();
const router = useRouter();
const isNew = () => route.name === "card-new";
const cardID = () => singleParam(route.params.id);
const backTo = () => (isNew() ? "/" : `/cards/${encodeURIComponent(cardID())}`);

const loading = ref(!isNew());
const saving = ref(false);
const error = ref("");
const statusEnabled = ref(false);
const initialFingerprint = ref("");
const fieldErrors = reactive<Record<string, string>>({});

const form = reactive<ActionCard>(emptyCard());
// Each action is edited as one command line (ADR-0012).
const primaryLine = ref("");
const statusLine = ref("");
const primaryEnv = ref<EnvironmentRow[]>([]);
const statusEnv = ref<EnvironmentRow[]>([]);
// Bumped on every route change so a response for a previous card is ignored.
let loadGeneration = 0;

function formFingerprint() {
	return fingerprint({
		form,
		statusEnabled: statusEnabled.value,
		primaryLine: primaryLine.value,
		statusLine: statusLine.value,
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
		// A colour outside the palette (edited file) shows and saves as None.
		color: knownCardColor(card.color),
		primary: copyAction(card.primary),
		status: card.status ? copyAction(card.status) : undefined,
	});
	statusEnabled.value = Boolean(card.status);
	primaryLine.value = commandLineOf(card.primary);
	statusLine.value = commandLineOf(card.status);
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
		env: fieldErrors[`${scope}Env`],
	};
}

function validate() {
	for (const key of Object.keys(fieldErrors)) delete fieldErrors[key];
	const result = validateCard({
		form,
		statusEnabled: statusEnabled.value,
		primaryLine: primaryLine.value,
		statusLine: statusLine.value,
	});
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
	const payload: ActionCard = {
		...form,
		id: isNew() ? "" : cardID(),
		name: form.name.trim(),
		primary: actionFrom(form.primary, primaryLine.value, primaryEnv.value),
		status: statusEnabled.value
			? actionFrom(form.status ?? emptyAction(), statusLine.value, statusEnv.value)
			: undefined,
	};
	try {
		const saved = isNew() ? await createCard(payload) : await updateCard(payload);
		applyCard(saved);
		notify.success(FEEDBACK.cardSaved, saved.name);
		// Stay in the edit context (UX spec §7.3); a new card moves to its own
		// edit route so a reload or a second save addresses the stored card.
		if (isNew()) {
			rememberSavedCard(saved);
			await router.replace(`/cards/${encodeURIComponent(saved.id)}/edit`);
		}
	} catch (saveError) {
		if (saveError instanceof ApiError && saveError.fields) applyServerErrors(saveError);
		else error.value = saveError instanceof Error ? saveError.message : FEEDBACK.unableToSave;
	} finally {
		saving.value = false;
	}
}

async function loadCard() {
	const generation = ++loadGeneration;
	error.value = "";
	if (isNew()) {
		applyCard(emptyCard());
		loading.value = false;
		return;
	}
	const saved = takeSavedCard(cardID());
	if (saved) {
		applyCard(saved);
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
		error.value = loadError instanceof Error ? loadError.message : FEEDBACK.unableToLoadCard;
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

useUnsavedChangesGuard(isDirty, () => saving.value);
</script>

<template>
	<main class="page">
		<PageHeader
			:eyebrow="isNew() ? 'New action card' : 'Edit action card'"
			:title="isNew() ? 'New card' : 'Edit card'"
			lede="Configure the identity, actions, and automatic checks for this card."
			:back-to="backTo()"
			:back-label="isNew() ? ACTIONS.backToOverview : ACTIONS.backToDetail"
		/>

		<div class="edit-feedback">
			<RequestState :message="error" tone="error" />
		</div>

		<div v-if="loading" class="loading-state" role="status" aria-live="polite">
			<span>{{ LOADING.card }}</span>
		</div>

		<Card v-else class="editor-card">
			<template #content>
				<CardIdentityFields
					v-model:name="form.name"
					v-model:description="form.description"
					v-model:icon="form.icon"
					v-model:color="form.color"
					:errors="{
						name: fieldErrors.name,
						description: fieldErrors.description,
						icon: fieldErrors.icon,
						color: fieldErrors.color,
					}"
				/>

				<ActionEditor
					v-model="form.primary"
					v-model:command-line="primaryLine"
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
					v-model:command-line="statusLine"
					v-model:environment="statusEnv"
					:model-value="form.status ?? emptyAction()"
					title="Status action"
					:errors="actionErrors('status')"
					@update:model-value="form.status = $event"
				/>

				<PollingFields
					v-model:polling-interval-seconds="form.pollingIntervalSeconds"
					v-model:fast-polling-interval-seconds="form.fastPollingIntervalSeconds"
					v-model:fast-polling-window-seconds="form.fastPollingWindowSeconds"
					class="polling-section"
					:errors="{
						pollingIntervalSeconds: fieldErrors.pollingIntervalSeconds,
						fastPollingIntervalSeconds: fieldErrors.fastPollingIntervalSeconds,
						fastPollingWindowSeconds: fieldErrors.fastPollingWindowSeconds,
					}"
				/>
			</template>
			<template #footer>
				<SaveBar :saving="saving" :cancel-to="backTo()" @save="save" />
			</template>
		</Card>
	</main>
</template>

<style scoped>
.edit-feedback {
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
	margin-bottom: var(--space-4);
}
.edit-feedback:empty {
	display: none;
}
.editor-card {
	max-width: 960px;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	box-shadow: var(--shadow-subtle);
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
</style>
