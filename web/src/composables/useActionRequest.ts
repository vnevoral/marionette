import { enqueuePrimary, enqueueStatus, type ActionCard, type StatusSnapshot } from "@/api";
import {
	expectsFollowUpCheck,
	waitBudgetMs,
	waitForNewerStatus,
} from "@/composables/useCardStatus";
import type { ActionKind, PendingPhase, RequestResult } from "@/types";
import { FEEDBACK } from "@/ui/vocabulary";

/** How a card action ended, independent of the view that shows it. */
export type ActionOutcome =
	| { kind: "accepted" }
	| { kind: "updated" }
	| { kind: "timeout" }
	| { kind: "aborted" }
	| { kind: "failed"; message: string };

export interface ActionRequestOptions {
	/** Aborting ends the wait with `aborted`; nothing is reported afterwards. */
	signal: AbortSignal;
	/** Called with `queued` before the request and `running` once a check is awaited. */
	onPhase(phase: PendingPhase): void;
	/** Every snapshot seen while waiting, so the view can show it. */
	onSnapshot?(snapshot: StatusSnapshot): void;
	/** A failed REST read while waiting; the wait continues. */
	onError?(): void;
}

/**
 * The one request flow behind "Run action" and "Check status" (UX spec §4):
 * queued → enqueue → accepted, or running → a check newer than the one the
 * server knew when it accepted the request (its 202 carries that baseline,
 * so a scheduled check that completed during the request does not count as
 * the result). Views map the outcome to their own wording with outcomeResult.
 */
export async function requestAction(
	card: ActionCard,
	action: ActionKind,
	options: ActionRequestOptions,
): Promise<ActionOutcome> {
	const { signal, onPhase, onSnapshot, onError } = options;
	onPhase("queued");
	let baseline: string | undefined;
	try {
		const accepted =
			action === "primary" ? await enqueuePrimary(card.id) : await enqueueStatus(card.id);
		baseline = accepted.checkedAt;
	} catch (error) {
		if (signal.aborted) return { kind: "aborted" };
		return {
			kind: "failed",
			message: error instanceof Error ? error.message : FEEDBACK.unableToQueue,
		};
	}
	if (signal.aborted) return { kind: "aborted" };
	if (!expectsFollowUpCheck(card, action)) return { kind: "accepted" };
	onPhase("running");
	const result = await waitForNewerStatus(card.id, baseline, {
		signal,
		maxWaitMs: waitBudgetMs(card, action),
		onSnapshot,
		onError,
	});
	return { kind: result };
}

/**
 * The feedback for an outcome; null when nothing should be reported. An
 * updated status is quiet: the badge already shows the new state.
 */
export function outcomeResult(
	outcome: ActionOutcome,
	labels: { accepted: string; updated: string },
): RequestResult | null {
	switch (outcome.kind) {
		case "accepted":
			return { tone: "success", message: labels.accepted };
		case "updated":
			return { tone: "success", message: labels.updated, quiet: true };
		case "timeout":
			return { tone: "error", message: FEEDBACK.resultUnavailable };
		case "failed":
			return { tone: "error", message: outcome.message };
		case "aborted":
			return null;
	}
}
