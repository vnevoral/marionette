<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import type { PairingCode } from "@/api";
import { ACCESS } from "@/ui/vocabulary";

// A freshly created pairing code for another device (FR-54): the code in
// large type, the time left and a link that fills the code in.
const props = defineProps<{ pairing: PairingCode }>();
const emit = defineEmits<{ renew: [] }>();

const now = ref(Date.now());
const timer = setInterval(() => (now.value = Date.now()), 1000);
onBeforeUnmount(() => clearInterval(timer));

const remainingMs = computed(() => Date.parse(props.pairing.expiresAt) - now.value);
const expired = computed(() => remainingMs.value <= 0);
const remaining = computed(() => {
	const seconds = Math.max(0, Math.ceil(remainingMs.value / 1000));
	return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
});
const link = computed(
	() => `${window.location.origin}/pair?code=${props.pairing.code.replace(/[^0-9A-Z]/g, "")}`,
);
// The clipboard API needs a secure context, which plain HTTP over a VPN is
// not; the read-only field still lets the link be selected and copied.
const canCopy = Boolean(window.isSecureContext && navigator.clipboard);
const copied = ref(false);

async function copyLink() {
	await navigator.clipboard.writeText(link.value);
	copied.value = true;
}
</script>

<template>
	<div class="pairing-code-panel">
		<template v-if="!expired">
			<p class="pairing-code-hint">{{ ACCESS.codeHint }}</p>
			<output class="pairing-code" aria-label="Pairing code">{{ pairing.code }}</output>
			<p class="pairing-code-expiry" aria-live="off">{{ ACCESS.codeExpiresIn(remaining) }}</p>
			<div class="pairing-link">
				<InputText
					:model-value="link"
					readonly
					aria-label="Pairing link"
					@focus="($event.target as HTMLInputElement).select()"
				/>
				<Button
					v-if="canCopy"
					type="button"
					:label="copied ? ACCESS.linkCopied : ACCESS.copyLink"
					icon="pi pi-copy"
					severity="secondary"
					outlined
					@click="copyLink"
				/>
			</div>
		</template>
		<p v-else class="pairing-code-hint">{{ ACCESS.codeExpired }}</p>
		<Button
			v-if="expired"
			type="button"
			:label="ACCESS.newCode"
			icon="pi pi-refresh"
			severity="secondary"
			outlined
			class="pairing-renew"
			@click="emit('renew')"
		/>
	</div>
</template>

<style scoped>
.pairing-code-panel {
	display: grid;
	gap: var(--space-3);
}
.pairing-code-hint,
.pairing-code-expiry {
	margin: 0;
	color: var(--color-muted);
}
.pairing-code {
	display: block;
	margin: 0;
	color: var(--color-ink);
	font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
	font-size: 2rem;
	font-weight: var(--font-weight-semibold);
	letter-spacing: 0.15em;
}
.pairing-link {
	display: flex;
	flex-wrap: wrap;
	gap: var(--space-2);
}
.pairing-link :deep(.p-inputtext) {
	flex: 1 1 14rem;
	min-width: 0;
}
.pairing-renew {
	justify-self: start;
}
</style>
