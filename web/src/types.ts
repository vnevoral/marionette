// Shared UI types. Vocabulary (labels, icons) lives in src/ui/vocabulary.ts;
// this module only holds the shapes the views and components agree on.

/** Semantic colour of a status badge or feedback line; never the only carrier of meaning. */
export type StatusTone = "healthy" | "problem" | "unknown" | "info" | "warning";

/** How a state is shown: text, icon and tone together (FR-25). */
export interface Presentation {
	label: string;
	icon: string;
	tone: StatusTone;
}

export type ActionKind = "primary" | "status";

/** Request state of a card action while it is in flight (UX spec §4). */
export type PendingPhase = "queued" | "running";

export interface PendingRequest {
	action: ActionKind;
	phase: PendingPhase;
}

/** Outcome shown briefly once a request has finished. */
export interface RequestResult {
	tone: "success" | "error";
	message: string;
	/** The badge already shows it; only assistive technology hears it (spec §4). */
	quiet?: boolean;
}

/** Repeatable editor rows carry a stable id so Vue keys survive removal. */
export type EnvironmentRow = { id: string; key: string; value: string };
