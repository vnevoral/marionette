import { onBeforeUnmount, ref } from "vue";
import type { RequestResult } from "@/types";

export type MessageTone = "info" | "success" | "error";

export const messageVisibleMs = 4000;

// One-line operation feedback: terminal outcomes stay visible briefly, pending
// notes (transient = false) are replaced by the outcome and never expire on
// their own.
export function useTransientMessage() {
	const feedback = ref<{ message: string; tone: MessageTone }>({ message: "", tone: "info" });
	let timer: ReturnType<typeof setTimeout> | undefined;

	function show(message: string, tone: MessageTone, transient = true) {
		clearTimeout(timer);
		timer = undefined;
		feedback.value = { message, tone };
		if (!transient) return;
		timer = setTimeout(() => {
			if (feedback.value.message === message) feedback.value = { message: "", tone: "info" };
		}, messageVisibleMs);
	}

	function clear() {
		show("", "info", false);
	}

	onBeforeUnmount(() => clearTimeout(timer));

	return { feedback, show, clear };
}

/** The outcome of the last card action, cleared after messageVisibleMs. */
export function useTransientResult() {
	const result = ref<RequestResult | null>(null);
	let timer: ReturnType<typeof setTimeout> | undefined;

	function show(next: RequestResult | null) {
		clearTimeout(timer);
		result.value = next;
		if (next) timer = setTimeout(() => (result.value = null), messageVisibleMs);
	}

	onBeforeUnmount(() => clearTimeout(timer));

	return { result, show };
}
