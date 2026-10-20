import { afterEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import { createFeedback } from "@/ui/feedback";
import MConfirmHost from "@/ui/MConfirmHost.vue";
import MToastHost from "@/ui/MToastHost.vue";

let wrapper: VueWrapper | undefined;

function mountHost(host: typeof MConfirmHost | typeof MToastHost) {
	const feedback = createFeedback();
	wrapper = mount(host, { attachTo: document.body, global: { plugins: [feedback] } });
	return feedback;
}

function dialog() {
	return document.querySelector<HTMLElement>('[role="alertdialog"]');
}

function button(name: string) {
	const found = Array.from(document.querySelectorAll("button")).find(
		(candidate) => candidate.textContent?.trim() === name,
	);
	if (!found) throw new Error(`button ${name} not rendered`);
	return found;
}

afterEach(() => {
	wrapper?.unmount();
	wrapper = undefined;
	document.body.innerHTML = "";
});

describe("MConfirmHost", () => {
	function ask(feedback: ReturnType<typeof createFeedback>) {
		const accept = vi.fn();
		const reject = vi.fn();
		feedback.confirm.require({
			header: "Delete card",
			message: 'Delete "Printer"?',
			acceptLabel: "Delete card",
			rejectLabel: "Cancel",
			danger: true,
			accept,
			reject,
		});
		return { accept, reject };
	}

	it("shows the question as an alert dialog with the focus on cancel", async () => {
		const feedback = mountHost(MConfirmHost);
		ask(feedback);
		await flushPromises();

		expect(dialog()?.textContent).toContain("Delete card");
		expect(dialog()?.textContent).toContain('Delete "Printer"?');
		expect(button("Delete card").className).toContain("m-button-danger");
		expect(document.activeElement).toBe(button("Cancel"));
	});

	it("accepts once and closes", async () => {
		const feedback = mountHost(MConfirmHost);
		const { accept, reject } = ask(feedback);
		await flushPromises();

		button("Delete card").click();
		await flushPromises();
		expect(accept).toHaveBeenCalledTimes(1);
		expect(reject).not.toHaveBeenCalled();
		expect(feedback.state.confirm).toBeNull();
	});

	it("rejects on cancel and on Escape", async () => {
		const feedback = mountHost(MConfirmHost);
		const first = ask(feedback);
		await flushPromises();
		button("Cancel").click();
		await flushPromises();
		expect(first.reject).toHaveBeenCalledTimes(1);
		expect(first.accept).not.toHaveBeenCalled();
		expect(feedback.state.confirm).toBeNull();

		const second = ask(feedback);
		await flushPromises();
		dialog()!.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
		await flushPromises();
		expect(second.reject).toHaveBeenCalledTimes(1);
		expect(feedback.state.confirm).toBeNull();
	});

	it("rejects an open question that a new one replaces", () => {
		const feedback = createFeedback();
		const first = ask(feedback);
		ask(feedback);
		expect(first.reject).toHaveBeenCalledTimes(1);
	});
});

describe("MToastHost", () => {
	it("shows the summary and detail in the tone and closes on request", async () => {
		const feedback = mountHost(MToastHost);
		feedback.toast.add({ tone: "success", summary: "Card saved", detail: "Lamp", life: 4000 });
		await flushPromises();

		const toast = document.querySelector(".m-toast");
		expect(toast?.className).toContain("m-toast-success");
		expect(toast?.querySelector(".m-toast-summary")?.textContent).toBe("Card saved");
		expect(toast?.querySelector(".m-toast-detail")?.textContent?.trim()).toBe("Lamp");
		expect(toast?.querySelector(".lucide-circle-check")).not.toBeNull();

		document.querySelector<HTMLButtonElement>('button[aria-label="Close"]')!.click();
		await flushPromises();
		expect(feedback.state.toasts).toEqual([]);
	});

	it("closes on its own after its life", async () => {
		vi.useFakeTimers();
		try {
			const feedback = mountHost(MToastHost);
			feedback.toast.add({ tone: "info", summary: "Saved", life: 4000 });
			await flushPromises();
			expect(feedback.state.toasts).toHaveLength(1);
			vi.advanceTimersByTime(4000);
			await flushPromises();
			expect(feedback.state.toasts).toEqual([]);
		} finally {
			vi.useRealTimers();
		}
	});
});
