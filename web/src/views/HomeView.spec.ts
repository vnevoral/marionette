import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount, RouterLinkStub } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import {
	enqueuePrimary,
	enqueueStatus,
	getStatus,
	listCards,
	type ActionCard,
	type StatusSnapshot,
} from "@/api";
import HomeView from "@/views/HomeView.vue";
import { FakeEventSource } from "@/test/fakeEventSource";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return {
		...actual,
		listCards: vi.fn(),
		getStatus: vi.fn(),
		enqueuePrimary: vi.fn(),
		enqueueStatus: vi.fn(),
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

const monitored: ActionCard = {
	id: "printer",
	name: "Printer",
	primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	status: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	pollingIntervalSeconds: 60,
	fastPollingIntervalSeconds: 10,
	fastPollingWindowSeconds: 120,
	currentStatus: snapshot(before),
};

// A status action without automatic checks: nothing runs after the primary
// action, so there is nothing to wait for.
const unpolled: ActionCard = {
	id: "sensor",
	name: "Sensor",
	primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	status: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	currentStatus: snapshot(before),
};

const plain: ActionCard = {
	id: "lamp",
	name: "Lamp",
	primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
};

function mountHome() {
	return mount(HomeView, {
		global: { plugins: [PrimeVue], stubs: { RouterLink: RouterLinkStub } },
	});
}

type Wrapper = ReturnType<ReturnType<typeof mountHome>["find"]>;

/** Visible text of the card's Last checked / outcome line. */
function noteText(card: Wrapper) {
	return card.find(".action-note .action-note-text").text();
}

function announcement(card: Wrapper) {
	return card.find(".action-note [role='status']").text();
}

function cardByName(wrapper: ReturnType<typeof mountHome>, name: string) {
	const card = wrapper.findAll(".action-card").find((candidate) => candidate.text().includes(name));
	if (!card) throw new Error(`card ${name} not rendered`);
	return card;
}

function buttonByLabel(
	scope: { findAll: (selector: string) => ReturnType<ReturnType<typeof mountHome>["findAll"]> },
	label: string,
) {
	const button = scope.findAll("button").find((candidate) => candidate.text().includes(label));
	if (!button) throw new Error(`button ${label} not rendered`);
	return button;
}

describe("HomeView", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
		vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "setInterval", "clearInterval"] });
		vi.mocked(listCards).mockReset().mockResolvedValue([monitored, unpolled, plain]);
		vi.mocked(getStatus).mockResolvedValue(snapshot(before));
		vi.mocked(enqueuePrimary).mockReset();
		vi.mocked(enqueueStatus).mockReset();
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it("releases the buttons after the action finishes and shows the result briefly (H1)", async () => {
		vi.mocked(enqueuePrimary).mockResolvedValue({
			cardId: "printer",
			actionKind: "primary",
			status: "accepted",
		});
		const wrapper = mountHome();
		await flushPromises();
		expect(FakeEventSource.instances).toHaveLength(1);
		const card = cardByName(wrapper, "Printer");
		expect(card.text()).toContain("Healthy");

		await buttonByLabel(card, "Run action").trigger("click");
		await flushPromises();
		expect(enqueuePrimary).toHaveBeenCalledWith("printer");
		expect(buttonByLabel(card, "Run action").attributes("disabled")).toBeDefined();
		expect(buttonByLabel(card, "Check status").attributes("disabled")).toBeDefined();
		expect(buttonByLabel(card, "Run action").classes()).toContain("p-button-loading");
		expect(card.text()).toContain("Running");

		FakeEventSource.last().status({ cardId: "printer", snapshot: snapshot(after, "fail") });
		await flushPromises();
		expect(buttonByLabel(card, "Run action").attributes("disabled")).toBeUndefined();
		expect(buttonByLabel(card, "Check status").attributes("disabled")).toBeUndefined();
		expect(buttonByLabel(card, "Run action").classes()).not.toContain("p-button-loading");
		// The badge shows the new state; the card keeps its Last checked line
		// and only announces the outcome (UX spec §4).
		expect(card.text()).toContain("Problem");
		expect(noteText(card)).toMatch(/^Last checked /);
		expect(announcement(card)).toBe("Updated");
		expect(card.find(".action-note").classes()).not.toContain("action-note-success");

		vi.advanceTimersByTime(4000);
		await flushPromises();
		expect(announcement(card)).toBe("");

		await buttonByLabel(card, "Check status").trigger("click");
		await flushPromises();
		expect(enqueueStatus).toHaveBeenCalledWith("printer");
		wrapper.unmount();
		expect(FakeEventSource.last().closed).toBe(true);
	});

	it("ignores a check the server already knew when it accepted the action", async () => {
		const during = "2026-09-26T10:00:02Z";
		vi.mocked(enqueuePrimary).mockResolvedValue({
			cardId: "printer",
			actionKind: "primary",
			status: "accepted",
			checkedAt: during,
		});
		const wrapper = mountHome();
		await flushPromises();
		const card = cardByName(wrapper, "Printer");
		await buttonByLabel(card, "Run action").trigger("click");
		await flushPromises();

		FakeEventSource.last().status({ cardId: "printer", snapshot: snapshot(during, "fail") });
		await flushPromises();
		expect(card.text()).toContain("Running");
		expect(buttonByLabel(card, "Run action").attributes("disabled")).toBeDefined();

		FakeEventSource.last().status({ cardId: "printer", snapshot: snapshot(after) });
		await flushPromises();
		expect(announcement(card)).toBe("Updated");
		expect(card.text()).toContain("Healthy");
		wrapper.unmount();
	});

	it("keeps the buttons enabled and shows the message when enqueue fails", async () => {
		vi.mocked(enqueuePrimary).mockRejectedValue(new Error("queue is full"));
		const wrapper = mountHome();
		await flushPromises();
		const card = cardByName(wrapper, "Printer");

		await buttonByLabel(card, "Run action").trigger("click");
		await flushPromises();
		expect(buttonByLabel(card, "Run action").attributes("disabled")).toBeUndefined();
		expect(buttonByLabel(card, "Run action").classes()).not.toContain("p-button-loading");
		expect(noteText(card)).toBe("queue is full");
		expect(card.find(".action-note").classes()).toContain("action-note-error");
		expect(card.text()).toContain("Healthy");
		wrapper.unmount();
	});

	it("reports acceptance immediately for cards without a status action", async () => {
		vi.mocked(enqueuePrimary).mockResolvedValue({
			cardId: "lamp",
			actionKind: "primary",
			status: "accepted",
		});
		const wrapper = mountHome();
		await flushPromises();
		const card = cardByName(wrapper, "Lamp");
		expect(card.findAll("button")).toHaveLength(1);

		await buttonByLabel(card, "Run action").trigger("click");
		await flushPromises();
		expect(buttonByLabel(card, "Run action").attributes("disabled")).toBeUndefined();
		expect(noteText(card)).toBe("Accepted");
		wrapper.unmount();
	});

	it("reports acceptance immediately for a monitored card without automatic checks", async () => {
		vi.mocked(enqueuePrimary).mockResolvedValue({
			cardId: "sensor",
			actionKind: "primary",
			status: "accepted",
		});
		const wrapper = mountHome();
		await flushPromises();
		const card = cardByName(wrapper, "Sensor");
		expect(card.findAll("button")).toHaveLength(2);

		await buttonByLabel(card, "Run action").trigger("click");
		await flushPromises();
		expect(buttonByLabel(card, "Run action").attributes("disabled")).toBeUndefined();
		expect(buttonByLabel(card, "Check status").attributes("disabled")).toBeUndefined();
		expect(noteText(card)).toBe("Accepted");
		expect(getStatus).not.toHaveBeenCalled();
		wrapper.unmount();
	});

	it("re-reads the card list instead of every status while the stream is down", async () => {
		const wrapper = mountHome();
		await flushPromises();
		expect(listCards).toHaveBeenCalledTimes(1);
		expect(getStatus).not.toHaveBeenCalled();

		vi.mocked(listCards).mockResolvedValue([
			{ ...monitored, currentStatus: snapshot(after, "fail") },
			unpolled,
			plain,
		]);
		vi.advanceTimersByTime(5000);
		await flushPromises();
		expect(listCards).toHaveBeenCalledTimes(2);
		expect(getStatus).not.toHaveBeenCalled();
		expect(cardByName(wrapper, "Printer").text()).toContain("Problem");

		FakeEventSource.last().open();
		await flushPromises();
		expect(listCards).toHaveBeenCalledTimes(3);
		vi.advanceTimersByTime(10000);
		await flushPromises();
		expect(listCards).toHaveBeenCalledTimes(3);
		wrapper.unmount();
	});

	it("marks monitored cards when the status cannot be refreshed and clears it on success", async () => {
		const wrapper = mountHome();
		await flushPromises();
		vi.mocked(listCards).mockRejectedValueOnce(new Error("Server unreachable"));
		vi.advanceTimersByTime(5000);
		await flushPromises();
		const printer = cardByName(wrapper, "Printer");
		expect(printer.find(".status-badge").text()).toBe("Unknown");
		expect(printer.find(".card-unavailable").text()).toBe("Status could not be refreshed");
		expect(cardByName(wrapper, "Lamp").find(".card-unavailable").exists()).toBe(false);

		vi.advanceTimersByTime(5000);
		await flushPromises();
		expect(cardByName(wrapper, "Printer").find(".status-badge").text()).toBe("Healthy");
		expect(cardByName(wrapper, "Printer").find(".card-unavailable").exists()).toBe(false);
		wrapper.unmount();
	});

	it("does not let a slow refresh overwrite a newer snapshot from the stream", async () => {
		const wrapper = mountHome();
		await flushPromises();
		let finishRefresh!: (cards: ActionCard[]) => void;
		vi.mocked(listCards).mockImplementationOnce(
			() => new Promise<ActionCard[]>((resolve) => (finishRefresh = resolve)),
		);
		vi.advanceTimersByTime(5000);
		await flushPromises();
		expect(listCards).toHaveBeenCalledTimes(2);

		FakeEventSource.last().status({ cardId: "printer", snapshot: snapshot(after, "fail") });
		await flushPromises();
		expect(cardByName(wrapper, "Printer").text()).toContain("Problem");

		finishRefresh([monitored, unpolled, plain]);
		await flushPromises();
		expect(cardByName(wrapper, "Printer").text()).toContain("Problem");
		wrapper.unmount();
	});
});
