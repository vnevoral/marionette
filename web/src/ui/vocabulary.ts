// Single source of UI wording (UX spec §3, §4, ADR-0007). Internal values such
// as "ok", "fail" or "accepted" never reach the screen; views and components
// look their presentation up here.
import type { Run, StatusState } from "@/api";
import type { PendingPhase, Presentation } from "@/types";

export const STATUS = {
	ok: { label: "Healthy", icon: "pi pi-check-circle", tone: "healthy" },
	fail: { label: "Problem", icon: "pi pi-exclamation-triangle", tone: "problem" },
	unknown: { label: "Unknown", icon: "pi pi-question-circle", tone: "unknown" },
} as const satisfies Record<StatusState, Presentation>;

export const NO_STATUS_CHECK: Presentation = {
	label: "No status check",
	icon: "pi pi-minus-circle",
	tone: "unknown",
};

/** Queued is amber with a queue icon, Running is blue with a spinner (spec §4). */
export const REQUEST = {
	queued: { label: "Queued", icon: "pi pi-hourglass", tone: "warning" },
	running: { label: "Running", icon: "pi pi-spin pi-spinner", tone: "info" },
} as const satisfies Record<PendingPhase, Presentation>;

export const OUTCOME = {
	ok: { label: "Successful", icon: "pi pi-check-circle", tone: "healthy" },
	fail: { label: "Failed", icon: "pi pi-times-circle", tone: "problem" },
	timeout: { label: "Timed out", icon: "pi pi-clock", tone: "warning" },
	canceled: { label: "Canceled", icon: "pi pi-ban", tone: "unknown" },
} as const satisfies Record<Run["outcome"], Presentation>;

/** Presentation of a device state; an unknown or missing value reads as Unknown. */
export function statusPresentation(state?: string): Presentation {
	return STATUS[(state ?? "unknown") as StatusState] ?? STATUS.unknown;
}

/** The dashboard note for a recorded primary run (FR-42a, UX spec §5.2). */
export function runOutcomeMessage(run: Pick<Run, "outcome" | "exitCode">): string {
	switch (run.outcome) {
		case "ok":
			return FEEDBACK.actionFinished;
		case "timeout":
			return FEEDBACK.actionTimedOut;
		case "canceled":
			return FEEDBACK.actionCanceled;
		default:
			return `Action failed · exit ${run.exitCode}`;
	}
}

export function outcomePresentation(outcome?: string): Presentation {
	return OUTCOME[(outcome ?? "fail") as Run["outcome"]] ?? OUTCOME.fail;
}

export const ACTIONS = {
	run: "Run action",
	check: "Check status",
	refresh: "Refresh",
	newCard: "New card",
	viewDetails: "View details",
	viewOutput: "View output",
	editCard: "Edit card",
	deleteCard: "Delete card",
	saveCard: "Save card",
	cancel: "Cancel",
	tryAgain: "Try again",
	backToOverview: "Back to overview",
	backToDetail: "Back to card detail",
	returnToOverview: "Return to overview",
} as const;

export const FEEDBACK = {
	accepted: "Accepted",
	actionAccepted: "Action accepted",
	updated: "Updated",
	statusUpdated: "Status updated",
	resultUnavailable: "Result not available yet",
	actionFinished: "Action finished",
	actionTimedOut: "Action timed out",
	actionCanceled: "Action canceled",
	actionsAsync: "Actions are queued asynchronously and may take a moment to report a new status.",
	statusUnavailable: "Status could not be refreshed",
	cardSaved: "Card saved",
	cardDeleted: "Card deleted",
	devicePaired: "Device paired",
	deviceRemoved: "Device removed",
	signedOut: "This device was removed",
	invalidCode: "The pairing code is invalid or has expired.",
	unableToPair: "Unable to pair this device",
	unableToLoadDevices: "Unable to load devices",
	unableToRemoveDevice: "Unable to remove device",
	unableToRenameDevice: "Unable to rename device",
	unableToCreateCode: "Unable to create a pairing code",
	deleting: "Deleting card...",
	unableToQueue: "Unable to queue action",
	unableToSave: "Unable to save card",
	unableToDelete: "Unable to delete card",
	unableToLoadCards: "Unable to load cards",
	unableToLoadCard: "Unable to load card",
	runsUnavailable: "Run history is unavailable",
	historyUnavailable: "Status history is unavailable",
} as const;

export const EMPTY = {
	cards: {
		title: "No action cards yet",
		body: "Create your first card to start monitoring and controlling a service.",
		action: "Create your first card",
	},
	runs: "No primary runs recorded yet.",
	history: "No status transitions recorded yet.",
	description: "No description provided.",
	statusNotConfigured: "Status monitoring is not configured for this card.",
	notChecked: "Not checked yet",
} as const;

export const LOADING = {
	cards: "Loading cards...",
	detail: "Loading card detail...",
	card: "Loading card...",
	runs: "Loading recent runs...",
	history: "Loading status history...",
	devices: "Loading devices...",
} as const;

/** Wording of the pairing screen and the Devices page (FR-50..FR-55). */
export const ACCESS = {
	pairTitle: "Pair this device",
	pairLede:
		"Enter a pairing code once. After that this browser opens Marionette without asking again.",
	bootstrapHint: "No device is paired yet. The pairing code is in the service log on the host:",
	bootstrapCommand: 'journalctl -u marionette | grep "pairing code"',
	otherDeviceHint: "Get a code on a paired device: Devices → Pair a new device.",
	codeLabel: "Pairing code",
	nameLabel: "Device name",
	pairAction: "Pair device",
	devicesTitle: "Devices",
	devicesLede: (days: number) =>
		`Browsers that can use Marionette. A device that is not used for ${days} days must pair again.`,
	thisDevice: "This device",
	pairedAt: "Paired",
	lastUsed: "Last used",
	pairNewDevice: "Pair a new device",
	newCode: "New code",
	codeHint: "Enter this code on the new device, or open the link there.",
	codeExpiresIn: (time: string) => `Valid for ${time}`,
	codeExpired: "The code has expired.",
	copyLink: "Copy link",
	linkCopied: "Link copied",
	removeDevice: "Remove device",
	removeMessage: (name: string, current: boolean) =>
		current
			? `Remove "${name}"? This is the device you are using: it will need a new pairing code to open Marionette again.`
			: `Remove "${name}"? It will need a new pairing code to open Marionette again.`,
	nowPaired: (name: string) => `"${name}" is now paired.`,
	removed: (name: string) => `"${name}" was removed.`,
	rename: "Rename",
	renameDevice: (name: string) => `Rename ${name}`,
	saveName: "Save",
	nameRequired: "Device name is required",
	renamed: (name: string) => `"${name}" saved.`,
} as const;
