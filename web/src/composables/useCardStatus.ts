import { ref, toValue, type MaybeRefOrGetter, type Ref } from "vue";
import { getStatus, type ActionCard, type StatusSnapshot } from "@/api";
import { useStatusEvents } from "@/composables/useStatusEvents";
import type { ActionKind } from "@/types";

export type WaitResult = "updated" | "timeout" | "aborted";

export interface WaitOptions {
	/** Aborting resolves the wait with `aborted`; nothing is written afterwards. */
	signal?: AbortSignal;
	/** Upper bound for the wait; resolves with `timeout` when exceeded. */
	maxWaitMs?: number;
	/** REST poll interval; SSE events short-circuit the wait whenever they arrive. */
	pollIntervalMs?: number;
}

export const defaultMaxWaitMs = 120_000;
export const defaultPollIntervalMs = 2000;

/** A snapshot counts as a new check when it carries a checkedAt later than
 * the previous one; an unchecked card has none, so any check is new. A check
 * older than the baseline (a late REST answer) is not. */
export function isNewerCheck(snapshot: StatusSnapshot, previousCheckedAt: string | undefined) {
	if (!snapshot.checkedAt) return false;
	if (!previousCheckedAt) return true;
	return Date.parse(snapshot.checkedAt) > Date.parse(previousCheckedAt);
}

/**
 * True when `next` may replace `current` in a view: an unchecked snapshot never
 * replaces a checked one, and a check is never replaced by an older one, so a
 * slow REST refresh cannot undo a snapshot that arrived through the stream in
 * the meantime.
 */
export function supersedes(next: StatusSnapshot, current: StatusSnapshot | undefined): boolean {
	if (!current?.checkedAt) return true;
	if (!next.checkedAt) return false;
	return Date.parse(next.checkedAt) >= Date.parse(current.checkedAt);
}

/**
 * Whether the server will run a status check after `action` is accepted, i.e.
 * whether a view should wait for a newer check. A status action always
 * schedules one; a primary action only activates fast polling, which the
 * scheduler does for cards with a status action and all three polling values
 * set (`Scheduler.NotifyPrimaryAction`). Otherwise the outcome is reported as
 * accepted right away.
 */
export function expectsFollowUpCheck(card: ActionCard, action: ActionKind): boolean {
	if (!card.status) return false;
	if (action === "status") return true;
	return (
		(card.pollingIntervalSeconds ?? 0) > 0 &&
		(card.fastPollingIntervalSeconds ?? 0) > 0 &&
		(card.fastPollingWindowSeconds ?? 0) > 0
	);
}

/** Allowance for the time a manual status check may spend queued behind
 * other actions before it starts. */
export const statusCheckQueueMarginMs = 30_000;

/** How often a view re-reads a card after an action: the card's fast polling
 * interval, or defaultPollIntervalMs when it has none (block 0050). */
export function runPollIntervalMs(card: ActionCard): number {
	const seconds = card.fastPollingIntervalSeconds ?? 0;
	return seconds > 0 ? seconds * 1000 : defaultPollIntervalMs;
}

/** How long a view waits for the run of an accepted primary action: its
 * timeout plus the queue margin. */
export function runWaitBudgetMs(card: ActionCard): number {
	return card.primary.timeoutSec * 1000 + statusCheckQueueMarginMs;
}

/**
 * How long a view waits for the follow-up check. A manual status check is
 * bounded by the status action's own timeout plus the queue margin; after a
 * primary action the check comes from fast polling, so the wait is the fast
 * polling window, or `defaultMaxWaitMs` when the card has none.
 */
export function waitBudgetMs(card: ActionCard, action: ActionKind): number {
	if (action === "status" && card.status) {
		return card.status.timeoutSec * 1000 + statusCheckQueueMarginMs;
	}
	return card.fastPollingWindowSeconds ? card.fastPollingWindowSeconds * 1000 : defaultMaxWaitMs;
}

interface WaitCallbacks {
	onSnapshot?(snapshot: StatusSnapshot): void;
	onError?(): void;
}

/**
 * Resolves once a status check newer than `previousCheckedAt` is visible for
 * the card. The server publishes `status.changed` only on state transitions,
 * so a check that keeps the same state is detected by the REST poll; a
 * transition arrives earlier through the shared stream.
 */
export function waitForNewerStatus(
	cardId: string,
	previousCheckedAt: string | undefined,
	options: WaitOptions & WaitCallbacks = {},
): Promise<WaitResult> {
	const {
		signal,
		maxWaitMs = defaultMaxWaitMs,
		pollIntervalMs = defaultPollIntervalMs,
		onSnapshot,
		onError,
	} = options;
	return new Promise((resolve) => {
		if (signal?.aborted) {
			resolve("aborted");
			return;
		}
		let settled = false;
		let inFlight = false;

		const consider = (snapshot: StatusSnapshot) => {
			if (settled) return;
			onSnapshot?.(snapshot);
			if (isNewerCheck(snapshot, previousCheckedAt)) finish("updated");
		};
		const unsubscribe = useStatusEvents().subscribe({
			onStatus(event) {
				if (event.cardId === cardId) consider(event.snapshot);
			},
		});
		const onAbort = () => finish("aborted");
		const deadline = setTimeout(() => finish("timeout"), maxWaitMs);
		const poll = setInterval(async () => {
			if (inFlight || settled) return;
			inFlight = true;
			try {
				consider(await getStatus(cardId));
			} catch {
				if (!settled) onError?.();
			} finally {
				inFlight = false;
			}
		}, pollIntervalMs);

		function finish(result: WaitResult) {
			if (settled) return;
			settled = true;
			unsubscribe();
			clearTimeout(deadline);
			clearInterval(poll);
			signal?.removeEventListener("abort", onAbort);
			resolve(result);
		}

		signal?.addEventListener("abort", onAbort);
	});
}

export interface CardStatus {
	snapshot: Ref<StatusSnapshot | null>;
	/** True after the last REST read for the card failed. */
	failed: Ref<boolean>;
	/** Replaces the snapshot from a freshly loaded card; `undefined` clears
	 * it. A snapshot older than the current one is ignored (see supersedes). */
	set(snapshot: StatusSnapshot | undefined): void;
	/** Applies a snapshot read for `id`; ignored when the composable meanwhile
	 * follows another card or already shows a newer check. */
	apply(id: string, snapshot: StatusSnapshot): void;
	/** Marks the last REST read for `id` as failed. */
	fail(id: string): void;
	/** Reads the snapshot over REST. */
	refresh(): Promise<void>;
}

/**
 * Tracks the status of one card through the shared stream. `enabled` should
 * be false for cards without a status action so no REST refresh is issued for
 * them. A change of `cardId` clears the snapshot and ignores stale responses.
 */
export function useCardStatus(
	cardId: MaybeRefOrGetter<string>,
	enabled: MaybeRefOrGetter<boolean> = true,
): CardStatus {
	const snapshot = ref<StatusSnapshot | null>(null);
	const failed = ref(false);

	function apply(id: string, next: StatusSnapshot) {
		if (toValue(cardId) !== id) return;
		if (supersedes(next, snapshot.value ?? undefined)) snapshot.value = next;
		failed.value = false;
	}

	function fail(id: string) {
		if (toValue(cardId) === id) failed.value = true;
	}

	async function refresh() {
		const id = toValue(cardId);
		if (!id || !toValue(enabled)) return;
		try {
			apply(id, await getStatus(id));
		} catch {
			fail(id);
		}
	}

	useStatusEvents().subscribe({
		onStatus(event) {
			apply(event.cardId, event.snapshot);
		},
		onRefresh() {
			void refresh();
		},
	});

	return {
		snapshot,
		failed,
		set(next) {
			if (!next) snapshot.value = null;
			else if (supersedes(next, snapshot.value ?? undefined)) snapshot.value = next;
			failed.value = false;
		},
		apply,
		fail,
		refresh,
	};
}
