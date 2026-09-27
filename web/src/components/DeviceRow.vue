<script setup lang="ts">
import { nextTick, ref, useId } from "vue";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import Tag from "primevue/tag";
import { ApiError, renameDevice, type Device } from "@/api";
import { formatDate } from "@/ui/format";
import { ACCESS, ACTIONS, FEEDBACK } from "@/ui/vocabulary";

// One paired device on the Devices page (FR-55) with in-place renaming
// (FR-57): the name turns into a field with Save / Cancel; Enter saves and
// Escape cancels.
const props = defineProps<{ device: Device }>();
const emit = defineEmits<{ remove: [device: Device]; renamed: [device: Device] }>();

const editing = ref(false);
const draft = ref("");
const nameError = ref("");
const saving = ref(false);
const inputId = useId();
const errorId = useId();
const nameInput = ref<{ $el: HTMLInputElement } | null>(null);
const renameButton = ref<{ $el: HTMLButtonElement } | null>(null);

async function startEditing() {
	draft.value = props.device.name;
	nameError.value = "";
	editing.value = true;
	await nextTick();
	nameInput.value?.$el.focus();
	nameInput.value?.$el.select();
}

async function stopEditing() {
	editing.value = false;
	nameError.value = "";
	await nextTick();
	renameButton.value?.$el.focus();
}

async function save() {
	if (saving.value) return;
	const name = draft.value.trim();
	if (!name) {
		nameError.value = ACCESS.nameRequired;
		return;
	}
	saving.value = true;
	nameError.value = "";
	try {
		const renamed = await renameDevice(props.device.id, name);
		emit("renamed", renamed);
		await stopEditing();
	} catch (failure) {
		if (failure instanceof ApiError && failure.fields?.name) nameError.value = failure.fields.name;
		else
			nameError.value = failure instanceof Error ? failure.message : FEEDBACK.unableToRenameDevice;
	} finally {
		saving.value = false;
	}
}
</script>

<template>
	<li class="device-row">
		<div class="device-identity">
			<i class="pi pi-desktop" aria-hidden="true" />
			<div class="device-body">
				<form v-if="editing" class="device-rename" novalidate @submit.prevent="save">
					<label :for="inputId" class="device-rename-label">{{ ACCESS.nameLabel }}</label>
					<div class="device-rename-controls">
						<InputText
							:id="inputId"
							ref="nameInput"
							v-model="draft"
							maxlength="64"
							autocomplete="off"
							:invalid="Boolean(nameError)"
							:aria-describedby="nameError ? errorId : undefined"
							@keydown.enter.prevent="save"
							@keydown.esc.prevent="stopEditing"
						/>
						<Button
							type="submit"
							:label="ACCESS.saveName"
							icon="pi pi-check"
							size="small"
							:loading="saving"
						/>
						<Button
							type="button"
							:label="ACTIONS.cancel"
							severity="secondary"
							outlined
							size="small"
							@click="stopEditing"
						/>
					</div>
					<small v-if="nameError" :id="errorId" class="field-error">{{ nameError }}</small>
				</form>
				<p v-else class="device-name">
					<span>{{ device.name }}</span>
					<Tag v-if="device.current" :value="ACCESS.thisDevice" severity="secondary" />
				</p>
				<p class="device-dates">
					{{ ACCESS.pairedAt }} {{ formatDate(device.pairedAt) }} · {{ ACCESS.lastUsed }}
					{{ formatDate(device.lastSeenAt) }}
				</p>
			</div>
		</div>
		<div v-if="!editing" class="device-actions">
			<Button
				ref="renameButton"
				icon="pi pi-pencil"
				text
				rounded
				severity="secondary"
				class="device-rename-button"
				:aria-label="ACCESS.renameDevice(device.name)"
				@click="startEditing"
			/>
			<Button
				icon="pi pi-trash"
				text
				rounded
				severity="danger"
				class="device-remove-button"
				:aria-label="`${ACCESS.removeDevice} ${device.name}`"
				@click="emit('remove', device)"
			/>
		</div>
	</li>
</template>

<style scoped>
.device-row {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: var(--space-3);
	padding: var(--space-3) 0;
	border-top: 1px solid var(--color-border);
}
.device-identity {
	display: flex;
	flex: 1;
	align-items: flex-start;
	gap: var(--space-3);
	min-width: 0;
}
.device-identity > i {
	margin-top: 0.2rem;
	color: var(--color-accent);
}
.device-body {
	flex: 1;
	min-width: 0;
}
.device-name {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: var(--space-2);
	margin: 0;
	color: var(--color-ink);
	font-weight: var(--font-weight-medium);
	overflow-wrap: anywhere;
}
.device-dates {
	margin: var(--space-1) 0 0;
	color: var(--color-muted);
	font-size: 0.85rem;
}
.device-actions {
	display: flex;
	flex-shrink: 0;
	gap: var(--space-1);
}
.device-rename {
	display: grid;
	gap: var(--space-1);
}
.device-rename-label {
	color: var(--color-ink);
	font-size: 0.85rem;
	font-weight: var(--font-weight-medium);
}
.device-rename-controls {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: var(--space-2);
}
.device-rename-controls > input {
	flex: 1 1 12rem;
	min-width: 0;
}
</style>
