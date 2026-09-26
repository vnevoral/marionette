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

export function outcomePresentation(outcome?: string): Presentation {
	return OUTCOME[(outcome ?? "fail") as Run["outcome"]] ?? OUTCOME.fail;
}

export const ACTIONS = {
	run: "Run action",
	check: "Check status",
	refresh: "Refresh",
	newCard: "New card",
	viewDetails: "View details",
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
	queued: "Queued",
	actionQueued: "Action queued",
	accepted: "Accepted",
	actionAccepted: "Action accepted",
	updated: "Updated",
	statusUpdated: "Status updated",
	resultUnavailable: "Result not available yet",
	statusUnavailable: "Status could not be refreshed",
	cardSaved: "Card saved",
	cardDeleted: "Card deleted",
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
} as const;
