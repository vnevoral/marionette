<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import { ApiError, pairDevice } from "@/api";
import InlineError from "@/components/InlineError.vue";
import PageHeader from "@/components/PageHeader.vue";
import { loadSession, setSession, useSession } from "@/composables/useSession";
import { useNotify } from "@/composables/useNotify";
import { safeNext } from "@/router/params";
import { suggestDeviceName } from "@/ui/deviceName";
import { ACCESS, FEEDBACK } from "@/ui/vocabulary";

// Pairing screen (FR-51..FR-53): one code, once per browser. The code may
// arrive in the link created on the Devices page (?code=).
const route = useRoute();
const router = useRouter();
const notify = useNotify();
const session = useSession();

const code = ref(typeof route.query.code === "string" ? formatTypedCode(route.query.code) : "");
const name = ref(suggestDeviceName(navigator.userAgent));
const error = ref("");
const nameError = ref("");
const pairing = ref(false);

const bootstrap = computed(() => session.value.status === "unpaired" && session.value.bootstrap);

/** Upper-cases and groups what is typed ("k7qm3x" → "K7QM-3X"). */
function formatTypedCode(value: string): string {
	const plain = value
		.toUpperCase()
		.replace(/[^0-9A-Z]/g, "")
		.slice(0, 8);
	return plain.length > 4 ? `${plain.slice(0, 4)}-${plain.slice(4)}` : plain;
}

function updateCode(value: string | undefined) {
	code.value = formatTypedCode(value ?? "");
}

async function pair() {
	error.value = "";
	nameError.value = "";
	if (!code.value || !name.value.trim()) {
		error.value = !code.value ? FEEDBACK.invalidCode : "";
		nameError.value = name.value.trim() ? "" : "Device name is required";
		return;
	}
	pairing.value = true;
	try {
		const device = await pairDevice(code.value, name.value.trim());
		setSession({ status: "paired", device, expiryDays: 0 });
		// Refresh the session for the expiry shown on the Devices page.
		void loadSession().catch(() => {});
		notify.success(FEEDBACK.devicePaired, device.name);
		await router.replace(safeNext(route.query.next));
	} catch (failure) {
		if (failure instanceof ApiError && failure.fields?.name) nameError.value = failure.fields.name;
		else if (failure instanceof ApiError && failure.status === 400)
			error.value = FEEDBACK.invalidCode;
		else error.value = failure instanceof Error ? failure.message : FEEDBACK.unableToPair;
	} finally {
		pairing.value = false;
	}
}

onMounted(() => {
	// The state may be stale (another tab paired, or the last device was
	// removed); the bootstrap hint depends on it.
	void loadSession().catch(() => {});
});
</script>

<template>
	<main class="page pair-page">
		<PageHeader eyebrow="Marionette access" :title="ACCESS.pairTitle" :lede="ACCESS.pairLede" />

		<section class="panel pair-panel" aria-labelledby="pair-form-title">
			<h2 id="pair-form-title" class="sr-only">{{ ACCESS.pairTitle }}</h2>
			<div v-if="bootstrap" class="pair-hint">
				<p>{{ ACCESS.bootstrapHint }}</p>
				<code class="pair-command">{{ ACCESS.bootstrapCommand }}</code>
			</div>
			<p v-else class="pair-hint pair-hint-muted">{{ ACCESS.otherDeviceHint }}</p>

			<form class="pair-form" novalidate @submit.prevent="pair">
				<label class="pair-field"
					>{{ ACCESS.codeLabel }}
					<InputText
						:model-value="code"
						class="pair-code"
						autocomplete="one-time-code"
						autocapitalize="characters"
						spellcheck="false"
						placeholder="XXXX-XXXX"
						maxlength="9"
						:invalid="Boolean(error)"
						@update:model-value="updateCode"
					/>
				</label>
				<label class="pair-field"
					>{{ ACCESS.nameLabel }}
					<InputText
						v-model="name"
						maxlength="64"
						autocomplete="off"
						:invalid="Boolean(nameError)"
					/>
					<small v-if="nameError" class="field-error">{{ nameError }}</small>
				</label>
				<InlineError :message="error" />
				<Button
					type="submit"
					:label="ACCESS.pairAction"
					icon="pi pi-link"
					class="primary-action-button pair-submit"
					:loading="pairing"
				/>
			</form>
		</section>
	</main>
</template>

<style scoped>
.pair-page {
	max-width: 560px;
}
/* minmax(0, 1fr): an input's intrinsic width (the monospace code field)
   must not widen the column past a 320 px screen with a wide fallback font. */
.pair-panel,
.pair-form,
.pair-field {
	grid-template-columns: minmax(0, 1fr);
}
.pair-panel {
	display: grid;
	gap: var(--space-6);
	padding: var(--space-6);
}
.pair-hint {
	display: grid;
	gap: var(--space-2);
	margin: 0;
}
.pair-hint-muted {
	color: var(--color-muted);
}
.pair-hint p {
	margin: 0;
}
.pair-command {
	display: block;
	overflow-x: auto;
	padding: var(--space-3);
	border-radius: var(--radius-sm);
	background: var(--color-canvas);
	color: var(--color-ink);
	font-size: 0.85rem;
	white-space: pre;
}
.pair-form {
	display: grid;
	gap: var(--space-4);
}
.pair-field {
	display: grid;
	gap: var(--space-2);
	color: var(--color-muted);
	font-size: 0.85rem;
	font-weight: var(--font-weight-medium);
}
.pair-code {
	font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
	font-size: 1.25rem;
	letter-spacing: 0.12em;
}
.pair-submit {
	justify-self: start;
}
</style>
