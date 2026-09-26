import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { h } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import { createMemoryHistory, createRouter } from "vue-router";
import PrimeVue from "primevue/config";
import {
	ApiError,
	deleteCard,
	enqueuePrimary,
	getCard,
	getRuns,
	getStatus,
	getStatusHistory,
	type ActionCard,
	type StatusSnapshot,
} from "@/api";
import CardDetailView from "@/views/CardDetailView.vue";
import { FEEDBACK } from "@/ui/vocabulary";
import { fakeConfirm } from "@/test/fakeConfirm";
import { fakeToast } from "@/test/fakeToast";
import { FakeEventSource } from "@/test/fakeEventSource";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return {
		...actual,
		getCard: vi.fn(),
		getRuns: vi.fn(),
		getStatusHistory: vi.fn(),
		getStatus: vi.fn(),
		enqueuePrimary: vi.fn(),
		enqueueStatus: vi.fn(),
		deleteCard: vi.fn(),
	};
});

const before = "2026-09-26T10:00:00Z";
const after = "2026-09-26T10:00:05Z";

function snapshot(checkedAt: string, state: StatusSnapshot["state"] = "ok"): StatusSnapshot {
	return {
		state,
		checkedAt,
		lastCheck: {
			actionKind: "status",
			startedAt: checkedAt,
			duration: 1,
			exitCode: 0,
			output: "",
			truncated: false,
			outcome: state === "ok" ? "ok" : "fail",
		},
	};
}

const cards: Record<string, ActionCard> = {
	printer: {
		id: "printer",
		name: "Printer",
		primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
		status: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
		pollingIntervalSeconds: 60,
		fastPollingIntervalSeconds: 10,
		fastPollingWindowSeconds: 120,
		currentStatus: snapshot(before),
	},
	sensor: {
		id: "sensor",
		name: "Sensor",
		primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
		status: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
		currentStatus: snapshot(before),
	},
	lamp: {
		id: "lamp",
		name: "Lamp",
		primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	},
};

const ASYNC_NOTE = FEEDBACK.actionsAsync;

/** Visible text of the Actions panel note (without the screen-reader region). */
function noteText(wrapper: ReturnType<typeof mount>) {
	return wrapper.find(".action-note .action-note-text").text();
}

function announcement(wrapper: ReturnType<typeof mount>) {
	return wrapper.find(".action-note [role='status']").text();
}

function buttonByLabel(wrapper: ReturnType<typeof mount>, label: string) {
	const button = wrapper.findAll("button").find((candidate) => candidate.text().includes(label));
	if (!button) throw new Error(`button ${label} not rendered`);
	return button;
}

async function mountDetail(id: string) {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{ path: "/", component: { render: () => h("div") } },
			{ path: "/cards/:id", name: "card-detail", component: CardDetailView },
		],
	});
	await router.push(`/cards/${id}`);
	await router.isReady();
	const confirm = fakeConfirm();
	const toast = fakeToast();
	const wrapper = mount(CardDetailView, {
		global: { plugins: [router, PrimeVue], provide: { ...confirm.provide, ...toast.provide } },
	});
	await flushPromises();
	return { wrapper, router, confirm, toast };
}

describe("CardDetailView", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
		vi.mocked(getCard)
			.mockReset()
			.mockImplementation(async (id) => {
				const card = cards[id];
				if (!card) throw new ApiError("card not found", 404);
				return card;
			});
		vi.mocked(deleteCard).mockReset().mockResolvedValue(undefined);
		vi.mocked(getRuns).mockReset().mockResolvedValue([]);
		vi.mocked(getStatusHistory).mockReset().mockResolvedValue([]);
		vi.mocked(enqueuePrimary).mockReset();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it("blocks the buttons until a newer check arrives, then refetches runs and history (H2)", async () => {
		vi.mocked(enqueuePrimary).mockResolvedValue({
			cardId: "printer",
			actionKind: "primary",
			status: "accepted",
		});
		const { wrapper } = await mountDetail("printer");
		expect(wrapper.find("h1").text()).toBe("Printer");
		expect(getRuns).toHaveBeenCalledTimes(1);
		expect(getStatusHistory).toHaveBeenCalledTimes(1);
		expect(wrapper.text()).toContain("Healthy");

		await buttonByLabel(wrapper, "Run action").trigger("click");
		await flushPromises();
		expect(enqueuePrimary).toHaveBeenCalledWith("printer");
		// The badge and the button spinner carry the pending state; the note
		// keeps its quiet text and only announces the phase (UX spec §4).
		expect(wrapper.find(".request-feedback").exists()).toBe(false);
		expect(noteText(wrapper)).toBe(ASYNC_NOTE);
		expect(announcement(wrapper)).toBe("Running");
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeDefined();
		expect(buttonByLabel(wrapper, "Check status").attributes("disabled")).toBeDefined();
		expect(wrapper.text()).toContain("Running");

		FakeEventSource.last().status({ cardId: "printer", snapshot: snapshot(after, "fail") });
		await flushPromises();
		expect(noteText(wrapper)).toBe(ASYNC_NOTE);
		expect(announcement(wrapper)).toBe("Status updated");
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeUndefined();
		expect(buttonByLabel(wrapper, "Check status").attributes("disabled")).toBeUndefined();
		expect(wrapper.text()).toContain("Problem");
		expect(getRuns).toHaveBeenCalledTimes(2);
		expect(getStatusHistory).toHaveBeenCalledTimes(2);
		wrapper.unmount();
		expect(FakeEventSource.last().closed).toBe(true);
	});

	it("keeps a newer stream snapshot when the loaded card carries an older check", async () => {
		vi.mocked(getCard).mockImplementationOnce(
			() =>
				new Promise<ActionCard>((resolve) => {
					setTimeout(() => resolve(cards.printer), 0);
				}),
		);
		const { wrapper } = await mountDetail("printer");
		FakeEventSource.last().status({ cardId: "printer", snapshot: snapshot(after, "fail") });
		await new Promise((resolve) => setTimeout(resolve, 0));
		await flushPromises();
		expect(wrapper.find("h1").text()).toBe("Printer");
		expect(wrapper.text()).toContain("Problem");
		wrapper.unmount();
	});

	it("marks the status as Unknown while it cannot be refreshed", async () => {
		vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "setInterval", "clearInterval"] });
		try {
			vi.mocked(getStatus).mockReset().mockRejectedValueOnce(new Error("Server unreachable"));
			const { wrapper } = await mountDetail("printer");
			expect(wrapper.find(".status-badge").text()).toBe("Healthy");
			vi.advanceTimersByTime(5000);
			await flushPromises();
			expect(wrapper.find(".status-badge").text()).toBe("Unknown");
			expect(wrapper.find(".summary-unavailable").text()).toBe("Status could not be refreshed");

			vi.mocked(getStatus).mockResolvedValue(snapshot(before));
			vi.advanceTimersByTime(5000);
			await flushPromises();
			expect(wrapper.find(".status-badge").text()).toBe("Healthy");
			expect(wrapper.find(".summary-unavailable").exists()).toBe(false);
			wrapper.unmount();
		} finally {
			vi.useRealTimers();
		}
	});

	it("shows the enqueue error and re-enables the buttons", async () => {
		vi.mocked(enqueuePrimary).mockRejectedValue(new Error("queue is full"));
		const { wrapper } = await mountDetail("printer");
		await buttonByLabel(wrapper, "Run action").trigger("click");
		await flushPromises();
		expect(noteText(wrapper)).toBe("queue is full");
		expect(wrapper.find(".action-note").classes()).toContain("action-note-error");
		expect(wrapper.find(".request-feedback").exists()).toBe(false);
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeUndefined();
		wrapper.unmount();
	});

	it("reloads when the route id changes and ignores the stale response", async () => {
		let resolvePrinter!: (card: ActionCard) => void;
		vi.mocked(getCard).mockImplementationOnce(
			() => new Promise<ActionCard>((resolve) => (resolvePrinter = resolve)),
		);
		const { wrapper, router } = await mountDetail("printer");
		expect(wrapper.text()).toContain("Loading card detail");

		await router.push("/cards/lamp");
		await flushPromises();
		expect(getCard).toHaveBeenLastCalledWith("lamp");
		expect(wrapper.find("h1").text()).toBe("Lamp");
		expect(wrapper.text()).toContain("No status check");

		resolvePrinter(cards.printer);
		await flushPromises();
		expect(wrapper.find("h1").text()).toBe("Lamp");
		expect(getRuns).toHaveBeenLastCalledWith("lamp");
		wrapper.unmount();
	});

	it("reports acceptance without waiting when the card has no status action", async () => {
		vi.mocked(enqueuePrimary).mockResolvedValue({
			cardId: "lamp",
			actionKind: "primary",
			status: "accepted",
		});
		const { wrapper } = await mountDetail("lamp");
		await buttonByLabel(wrapper, "Run action").trigger("click");
		await flushPromises();
		expect(noteText(wrapper)).toBe("Action accepted");
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeUndefined();
		expect(getRuns).toHaveBeenCalledTimes(2);
		wrapper.unmount();
	});

	it("reports acceptance without waiting when no automatic check follows the primary action", async () => {
		vi.mocked(enqueuePrimary).mockResolvedValue({
			cardId: "sensor",
			actionKind: "primary",
			status: "accepted",
		});
		vi.mocked(getStatus).mockReset();
		const { wrapper } = await mountDetail("sensor");
		await buttonByLabel(wrapper, "Run action").trigger("click");
		await flushPromises();
		expect(noteText(wrapper)).toBe("Action accepted");
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeUndefined();
		expect(buttonByLabel(wrapper, "Check status").attributes("disabled")).toBeUndefined();
		expect(getStatus).not.toHaveBeenCalled();
		wrapper.unmount();
	});

	it("distinguishes a missing card from an unreachable server", async () => {
		const missing = await mountDetail("missing");
		expect(missing.wrapper.text()).toContain("Card not found");
		expect(missing.wrapper.text()).toContain('"missing"');
		expect(missing.wrapper.text()).not.toContain("Try again");
		missing.wrapper.unmount();
		vi.mocked(getCard).mockClear();

		vi.mocked(getCard).mockRejectedValueOnce(new ApiError("Server unreachable", 0));
		const { wrapper } = await mountDetail("printer");
		expect(wrapper.text()).toContain("Card unavailable");
		expect(wrapper.text()).toContain("Server unreachable");
		await buttonByLabel(wrapper, "Try again").trigger("click");
		await flushPromises();
		expect(getCard).toHaveBeenCalledTimes(2);
		expect(wrapper.find("h1").text()).toBe("Printer");
		wrapper.unmount();
	});

	it("deletes the card only after the dialog names it and is accepted", async () => {
		const { wrapper, router, confirm, toast } = await mountDetail("printer");
		await buttonByLabel(wrapper, "Delete card").trigger("click");
		expect(deleteCard).not.toHaveBeenCalled();
		expect(confirm.last().header).toBe("Delete card");
		expect(confirm.last().message).toContain('"Printer"');
		expect(confirm.last().acceptLabel).toBe("Delete card");

		confirm.last().accept?.();
		await flushPromises();
		expect(deleteCard).toHaveBeenCalledWith("printer");
		expect(router.currentRoute.value.path).toBe("/");
		expect(toast.summaries()).toEqual(["Card deleted"]);
		expect(toast.add.mock.calls[0][0].detail).toContain('"Printer"');
		wrapper.unmount();
	});

	it("keeps the detail and shows the error inline when deletion fails, without a toast", async () => {
		vi.mocked(deleteCard).mockRejectedValue(new Error("Server unreachable"));
		const { wrapper, router, confirm, toast } = await mountDetail("printer");
		await buttonByLabel(wrapper, "Delete card").trigger("click");
		confirm.last().accept?.();
		await flushPromises();
		expect(router.currentRoute.value.path).toBe("/cards/printer");
		expect(wrapper.find(".request-feedback").text()).toBe("Server unreachable");
		expect(toast.add).not.toHaveBeenCalled();
		wrapper.unmount();
	});

	it("clears a finished action message after a few seconds", async () => {
		vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "setInterval", "clearInterval"] });
		try {
			vi.mocked(enqueuePrimary).mockRejectedValue(new Error("queue is full"));
			const { wrapper } = await mountDetail("printer");
			await buttonByLabel(wrapper, "Run action").trigger("click");
			await flushPromises();
			expect(noteText(wrapper)).toBe("queue is full");
			vi.advanceTimersByTime(4000);
			await flushPromises();
			expect(noteText(wrapper)).toBe(ASYNC_NOTE);
			wrapper.unmount();
		} finally {
			vi.useRealTimers();
		}
	});
});
