import { ref, type Ref } from "vue";
import { getRuns, getStatusHistory, type Run, type StatusChange } from "@/api";
import { FEEDBACK } from "@/ui/vocabulary";

// Recent runs and status history of one card. reset() starts a new generation
// so responses for a previous card or an older load are ignored.
export function useCardActivity(cardID: Ref<string>) {
	const runs = ref<Run[]>([]);
	const history = ref<StatusChange[]>([]);
	const runsLoading = ref(true);
	const historyLoading = ref(true);
	const errors = ref<{ runs?: string; history?: string }>({});
	let generation = 0;

	function reset() {
		generation++;
		runs.value = [];
		history.value = [];
		runsLoading.value = true;
		historyLoading.value = true;
		errors.value = {};
	}

	async function loadRuns(current: number) {
		try {
			const loaded = await getRuns(cardID.value);
			if (current !== generation) return;
			runs.value = loaded ?? [];
			delete errors.value.runs;
		} catch {
			if (current === generation) errors.value.runs = FEEDBACK.runsUnavailable;
		} finally {
			if (current === generation) runsLoading.value = false;
		}
	}

	async function loadHistory(current: number) {
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

	return { runs, history, runsLoading, historyLoading, errors, reset, load };
}
