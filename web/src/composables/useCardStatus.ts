import { ref, toValue, type MaybeRefOrGetter, type Ref } from "vue";
import { getStatus, type StatusSnapshot } from "@/api";
import { useStatusEvents } from "@/composables/useStatusEvents";

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

const zeroTimePrefix = "0001-01-01";

/** A snapshot counts as a new check when it carries a real, different checkedAt. */
export function isNewerCheck(snapshot: StatusSnapshot, previousCheckedAt: string | undefined) {
	if (!snapshot.checkedAt || snapshot.checkedAt.startsWith(zeroTimePrefix)) return false;
	return snapshot.checkedAt !== previousCheckedAt;
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
	/** Replaces the snapshot, e.g. from a freshly loaded card. */
	set(snapshot: StatusSnapshot | undefined): void;
	/** Reads the snapshot over REST. */
	refresh(): Promise<void>;
	waitForNewer(previousCheckedAt: string | undefined, options?: WaitOptions): Promise<WaitResult>;
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
		snapshot.value = next;
		failed.value = false;
	}

	async function refresh() {
		const id = toValue(cardId);
		if (!id || !toValue(enabled)) return;
		try {
			apply(id, await getStatus(id));
		} catch {
			if (toValue(cardId) === id) failed.value = true;
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
			snapshot.value = next ?? null;
			failed.value = false;
		},
		refresh,
		waitForNewer(previousCheckedAt, options) {
			const id = toValue(cardId);
			return waitForNewerStatus(id, previousCheckedAt, {
				...options,
				onSnapshot: (next) => apply(id, next),
				onError: () => {
					if (toValue(cardId) === id) failed.value = true;
				},
			});
		},
	};
}
