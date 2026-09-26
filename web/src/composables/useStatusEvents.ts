import { computed, getCurrentScope, onScopeDispose, readonly, ref, type Ref } from "vue";
import { connectStatusEvents, type StatusEvent } from "@/api";

export interface StatusListener {
	/** Receives every `status.changed` event from the shared stream. */
	onStatus(event: StatusEvent): void;
	/**
	 * Called on each polling tick while the stream is down and once when it
	 * (re)connects, so a view can catch up on checks it may have missed.
	 */
	onRefresh?(): void;
}

export type StatusEventsMode = "live" | "polling";

export type StatusEventsConnector = (
	onStatus: (event: StatusEvent) => void,
) => EventSource | undefined;

export interface StatusEvents {
	/** True while the SSE stream is open. */
	connected: Readonly<Ref<boolean>>;
	/** `live` while connected, otherwise `polling` (REST refresh every interval). */
	mode: Readonly<Ref<StatusEventsMode>>;
	/** Registers a listener and returns its unsubscribe function. */
	subscribe(listener: StatusListener): () => void;
	/** Number of active listeners; the stream is open only while it is > 0. */
	readonly subscribers: number;
}

export const pollingIntervalMs = 5000;

/**
 * Builds a reference-counted status stream: the first subscriber opens one
 * EventSource, the last unsubscribe closes it. While the stream is not open
 * (before the first `open`, after an `error`, or without EventSource support)
 * listeners are asked to refresh over REST every `intervalMs`.
 */
export function createStatusEvents(
	connect: StatusEventsConnector = connectStatusEvents,
	intervalMs = pollingIntervalMs,
): StatusEvents {
	const listeners = new Set<StatusListener>();
	const connected = ref(false);
	let source: EventSource | undefined;
	let pollTimer: ReturnType<typeof setInterval> | undefined;

	function refreshAll() {
		for (const listener of listeners) listener.onRefresh?.();
	}

	function startPolling() {
		if (pollTimer !== undefined) return;
		pollTimer = setInterval(refreshAll, intervalMs);
	}

	function stopPolling() {
		if (pollTimer === undefined) return;
		clearInterval(pollTimer);
		pollTimer = undefined;
	}

	function open() {
		source = connect((event) => {
			for (const listener of listeners) listener.onStatus(event);
		});
		startPolling();
		if (!source) return;
		source.addEventListener("open", () => {
			connected.value = true;
			stopPolling();
			refreshAll();
		});
		source.addEventListener("error", () => {
			connected.value = false;
			startPolling();
		});
	}

	function close() {
		source?.close();
		source = undefined;
		stopPolling();
		connected.value = false;
	}

	function subscribe(listener: StatusListener) {
		listeners.add(listener);
		if (listeners.size === 1) open();
		let active = true;
		return () => {
			if (!active) return;
			active = false;
			listeners.delete(listener);
			if (listeners.size === 0) close();
		};
	}

	return {
		connected: readonly(connected),
		mode: computed(() => (connected.value ? "live" : "polling")),
		subscribe,
		get subscribers() {
			return listeners.size;
		},
	};
}

const shared = createStatusEvents();

/**
 * Shared access to the status stream. Subscriptions made inside a component
 * or effect scope are released automatically when that scope is disposed.
 */
export function useStatusEvents(): StatusEvents {
	return {
		connected: shared.connected,
		mode: shared.mode,
		subscribe(listener) {
			const unsubscribe = shared.subscribe(listener);
			if (getCurrentScope()) onScopeDispose(unsubscribe);
			return unsubscribe;
		},
		get subscribers() {
			return shared.subscribers;
		},
	};
}
