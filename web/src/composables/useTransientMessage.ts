import { onBeforeUnmount, ref } from "vue";

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
