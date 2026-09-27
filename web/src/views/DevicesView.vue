<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import Button from "primevue/button";
import ProgressSpinner from "primevue/progressspinner";
import { useConfirm } from "primevue/useconfirm";
import { createPairingCode, listDevices, removeDevice, type Device, type PairingCode } from "@/api";
import DetailPanel from "@/components/DetailPanel.vue";
import DeviceRow from "@/components/DeviceRow.vue";
import EmptyState from "@/components/EmptyState.vue";
import InlineError from "@/components/InlineError.vue";
import PageHeader from "@/components/PageHeader.vue";
import PairingCodePanel from "@/components/PairingCodePanel.vue";
import RequestState from "@/components/RequestState.vue";
import { useNotify } from "@/composables/useNotify";
import { markUnpaired, setSession, useSession } from "@/composables/useSession";
import { useTransientMessage } from "@/composables/useTransientMessage";
import { ACCESS, ACTIONS, FEEDBACK, LOADING } from "@/ui/vocabulary";

// Paired devices (FR-54, FR-55, FR-57): list, rename, remove, and a code for
// another device.
const router = useRouter();
const confirm = useConfirm();
const notify = useNotify();
const session = useSession();
const { feedback, show: showMessage } = useTransientMessage();

const devices = ref<Device[]>([]);
const loading = ref(true);
const error = ref("");
const pairing = ref<PairingCode | null>(null);
const creatingCode = ref(false);
let watchTimer: ReturnType<typeof setInterval> | undefined;

const accessDisabled = computed(() => session.value.status === "open");
const expiryDays = computed(() =>
	session.value.status === "paired" && session.value.expiryDays ? session.value.expiryDays : 60,
);

async function load() {
	error.value = "";
	try {
		devices.value = await listDevices();
	} catch (failure) {
		error.value = failure instanceof Error ? failure.message : FEEDBACK.unableToLoadDevices;
	} finally {
		loading.value = false;
	}
}

function stopWatching() {
	clearInterval(watchTimer);
	watchTimer = undefined;
}

// While a code is shown, look for the device that uses it so the page can
// say so without a manual refresh.
function watchForNewDevice() {
	stopWatching();
	const known = new Set(devices.value.map((device) => device.id));
	watchTimer = setInterval(async () => {
		if (pairing.value && Date.parse(pairing.value.expiresAt) <= Date.now()) return stopWatching();
		try {
			const latest = await listDevices();
			devices.value = latest;
			const added = latest.find((device) => !known.has(device.id));
			if (added) {
				pairing.value = null;
				stopWatching();
				showMessage(ACCESS.nowPaired(added.name), "success");
			}
		} catch {
			// The next tick retries.
		}
	}, 3000);
}

async function newCode() {
	creatingCode.value = true;
	error.value = "";
	try {
		pairing.value = await createPairingCode();
		watchForNewDevice();
	} catch (failure) {
		error.value = failure instanceof Error ? failure.message : FEEDBACK.unableToCreateCode;
	} finally {
		creatingCode.value = false;
	}
}

function confirmRemove(device: Device) {
	confirm.require({
		header: ACCESS.removeDevice,
		message: ACCESS.removeMessage(device.name, device.current),
		icon: "pi pi-exclamation-triangle",
		acceptLabel: ACCESS.removeDevice,
		rejectLabel: ACTIONS.cancel,
		acceptProps: { severity: "danger" },
		rejectProps: { severity: "secondary", outlined: true },
		defaultFocus: "reject",
		accept: () => void remove(device),
	});
}

async function remove(device: Device) {
	try {
		await removeDevice(device.id);
	} catch (failure) {
		showMessage(
			failure instanceof Error ? failure.message : FEEDBACK.unableToRemoveDevice,
			"error",
		);
		return;
	}
	if (device.current) {
		// Removing this browser signs it out; the notice travels with the
		// navigation to the pairing screen (UX spec §4).
		markUnpaired();
		await router.replace({ name: "pair" });
		notify.success(FEEDBACK.signedOut, device.name);
		return;
	}
	devices.value = devices.value.filter((candidate) => candidate.id !== device.id);
	showMessage(ACCESS.removed(device.name), "success");
}

function renamed(device: Device) {
	devices.value = devices.value.map((candidate) =>
		candidate.id === device.id ? device : candidate,
	);
	if (device.current && session.value.status === "paired") {
		setSession({ ...session.value, device });
	}
	showMessage(ACCESS.renamed(device.name), "success");
}

onMounted(() => {
	if (accessDisabled.value) loading.value = false;
	else void load();
});
onBeforeUnmount(stopWatching);
</script>

<template>
	<main class="page">
		<PageHeader
			eyebrow="Marionette access"
			:title="ACCESS.devicesTitle"
			:lede="accessDisabled ? undefined : ACCESS.devicesLede(expiryDays)"
		>
			<template v-if="!accessDisabled" #actions>
				<Button
					:label="ACCESS.pairNewDevice"
					icon="pi pi-plus"
					class="primary-action-button"
					:loading="creatingCode"
					@click="newCode"
				/>
			</template>
		</PageHeader>

		<EmptyState
			v-if="accessDisabled"
			icon="pi pi-lock-open"
			title="Access control is disabled"
			body="This server runs with MARIONETTE_AUTH=off: every browser that reaches it can use Marionette and no devices are paired."
		/>

		<template v-else>
			<RequestState class="devices-feedback" :message="feedback.message" :tone="feedback.tone" />
			<InlineError :message="error" />

			<DetailPanel
				v-if="pairing"
				:title="ACCESS.pairNewDevice"
				icon="pi pi-link"
				class="devices-code"
			>
				<PairingCodePanel :pairing="pairing" @renew="newCode" />
			</DetailPanel>

			<div v-if="loading" class="loading-state" role="status" aria-live="polite">
				<ProgressSpinner :aria-label="LOADING.devices" />
				<span>{{ LOADING.devices }}</span>
			</div>

			<DetailPanel v-else :title="ACCESS.devicesTitle" :count="devices.length">
				<ul class="device-list">
					<DeviceRow
						v-for="device in devices"
						:key="device.id"
						:device="device"
						@remove="confirmRemove"
						@renamed="renamed"
					/>
				</ul>
			</DetailPanel>
		</template>
	</main>
</template>

<style scoped>
.devices-feedback,
.devices-code {
	margin-bottom: var(--space-4);
}
.device-list {
	display: grid;
	gap: 0;
	margin: 0;
	padding: 0;
	list-style: none;
}
</style>
