import { onBeforeUnmount, ref, watch, type Ref } from "vue";
import { getRuns, getStatusHistory, type ActionCard, type Run, type StatusChange } from "@/api";
import { runPollIntervalMs, runWaitBudgetMs } from "@/composables/useCardStatus";
import { useStatusEvents } from "@/composables/useStatusEvents";
import { FEEDBACK } from "@/ui/vocabulary";

export type RunWaitResult = "found" | "timeout" | "aborted";

/**
 * What was known about the runs before an action: the newest run's start
 * (undefined for a card without runs), or null when the list was still
 * loading or its last read failed, so no run can be told apart as new.
 */
export type RunBaseline = { startedAt: string | undefined } | null;

/** True when `run` started after the newest run known before the action. */
function isNewerRun(run: Run | undefined, baseline: RunBaseline): boolean {
	if (!run || !baseline) return false;
	if (!baseline.startedAt) return true;
	return Date.parse(run.startedAt) > Date.parse(baseline.startedAt);
}

/** The wait of waitForNewRun, so a stream event or another read can end it. */
interface RunWait {
	baseline: RunBaseline;
	controller: AbortController;
	found(): void;
}

// Recent runs and status history of one card. reset() starts a new generation
// so responses for a previous card or an older load are ignored. A recorded
// run of the card (`run.recorded`, FR-42a) re-reads the runs at once, and so
// does a reconnect of the stream, which may have missed one.
export function useCardActivity(cardID: Ref<string>) {
	const runs = ref<Run[]>([]);
	const history = ref<StatusChange[]>([]);
	const runsLoading = ref(true);
	const historyLoading = ref(true);
	const errors = ref<{ runs?: string; history?: string }>({});
	let generation = 0;
	let runWait: RunWait | undefined;
	const events = useStatusEvents();
	// The runs list reflects the server: its last read for this card succeeded.
	let runsCurrent = false;

	function reset() {
		generation++;
		stopWaitingForRun();
		runsCurrent = false;
		runs.value = [];
		history.value = [];
		runsLoading.value = true;
		historyLoading.value = true;
		errors.value = {};
	}

	/** Reads the runs; true when the answer still belongs to `current`. */
	async function loadRuns(current: number): Promise<boolean> {
		try {
			const loaded = await getRuns(cardID.value);
			if (current !== generation) return false;
			runs.value = loaded ?? [];
			runsCurrent = true;
			delete errors.value.runs;
			if (runWait && isNewerRun(runs.value[0], runWait.baseline)) runWait.found();
			return true;
		} catch {
			if (current !== generation) return false;
			errors.value.runs = FEEDBACK.runsUnavailable;
			runsCurrent = false;
			return false;
		} finally {
			if (current === generation) runsLoading.value = false;
		}
	}

	async function loadHistory(current = generation) {
		try {
			const loaded = await getStatusHistory(cardID.value);
			if (current !== generation) return;
			history.value = loaded ?? [];
			delete errors.value.history;
		} catch {
			if (current === generation) errors.value.history = FEEDBACK.historyUnavailable;
		} finally {
			if (current === generation) historyLoading.value = false;
		}
	}

	/** Fetches both lists for the current generation; each failure is reported separately. */
	function load() {
		const current = generation;
		return Promise.allSettled([loadRuns(current), loadHistory(current)]);
	}

	/** The baseline for waitForNewRun, taken before the action is sent. */
	function runBaseline(): RunBaseline {
		return runsCurrent ? { startedAt: runs.value[0]?.startedAt } : null;
	}

	function stopWaitingForRun() {
		runWait?.controller.abort();
		runWait = undefined;
	}

	events.subscribe({
		onRun(event) {
			if (event.cardId !== cardID.value) return;
			// Without a baseline any run recorded during the wait is the new one.
			if (runWait && (!runWait.baseline || isNewerRun(event.run, runWait.baseline)))
				runWait.found();
			void loadRuns(generation);
		},
	});

	let disconnected = false;
	watch(events.connected, (connected) => {
		if (!connected) disconnected = true;
		else if (disconnected) {
			disconnected = false;
			void loadRuns(generation);
		}
	});

	/**
	 * After an accepted primary action (blocks 0050, 0058): waits until a run
	 * newer than the newest one known before the action appears, or the
	 * action's timeout plus the queue margin passes. The `run.recorded` event
	 * normally ends it; REST reads at the card's fast polling interval (2 s
	 * without one) are the fallback. While the stream is live the first read
	 * waits for the action's own timeout, since the event comes first when
	 * the run ends earlier; it is brought forward when the stream drops.
	 * Comparing server timestamps keeps a skewed browser clock out of it.
	 * Without a baseline (the list was loading or failed, see runBaseline) a
	 * read cannot tell the new run apart, so only the event ends the wait
	 * early. A new wait, reset() or unmounting ends the previous one; a failed
	 * read is shown and the wait goes on.
	 */
	function waitForNewRun(card: ActionCard, baseline: RunBaseline) {
		stopWaitingForRun();
		const controller = new AbortController();
		const current = generation;
		const intervalMs = runPollIntervalMs(card);

		return new Promise<RunWaitResult>((resolve) => {
			let timer: ReturnType<typeof setTimeout> | undefined;
			let settled = false;
			const finish = (result: RunWaitResult) => {
				if (settled) return;
				settled = true;
				clearTimeout(timer);
				clearTimeout(deadline);
				stopWatchingStream();
				controller.signal.removeEventListener("abort", onAbort);
				if (runWait?.controller === controller) runWait = undefined;
				resolve(result);
			};
			const onAbort = () => finish("aborted");
			const schedule = (delayMs: number) => {
				clearTimeout(timer);
				timer = setTimeout(poll, delayMs);
			};
			const poll = async () => {
				timer = undefined;
				await loadRuns(current);
				if (!settled) schedule(intervalMs);
			};
			let quiet = events.connected.value;
			const stopWatchingStream = watch(events.connected, (connected) => {
				if (connected || !quiet) return;
				quiet = false;
				if (timer !== undefined) schedule(intervalMs);
			});
			const deadline = setTimeout(() => finish("timeout"), runWaitBudgetMs(card));
			runWait = { baseline, controller, found: () => finish("found") };
			controller.signal.addEventListener("abort", onAbort);
			schedule(quiet ? Math.max(intervalMs, card.primary.timeoutSec * 1000) : intervalMs);
		});
	}

	onBeforeUnmount(stopWaitingForRun);

	return {
		runs,
		history,
		runsLoading,
		historyLoading,
		errors,
		reset,
		load,
		loadHistory,
		runBaseline,
		waitForNewRun,
	};
}
