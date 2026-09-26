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
		vi.mocked(listCards).mockResolvedValue([monitored, plain]);
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
		expect(card.find(".request-feedback").text()).toBe("Updated");
		expect(card.find(".request-feedback").classes()).toContain("request-feedback-success");
		expect(card.text()).toContain("Problem");

		vi.advanceTimersByTime(4000);
		await flushPromises();
		expect(card.find(".request-feedback").exists()).toBe(false);

		await buttonByLabel(card, "Check status").trigger("click");
		await flushPromises();
		expect(enqueueStatus).toHaveBeenCalledWith("printer");
		wrapper.unmount();
		expect(FakeEventSource.last().closed).toBe(true);
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
		expect(card.find(".request-feedback").text()).toBe("queue is full");
		expect(card.find(".request-feedback").classes()).toContain("request-feedback-error");
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
		expect(card.find(".request-feedback").text()).toBe("Accepted");
		wrapper.unmount();
	});

	it("falls back to REST polling of monitored cards while the stream is down", async () => {
		const wrapper = mountHome();
		await flushPromises();
		expect(getStatus).toHaveBeenCalledTimes(1);
		vi.advanceTimersByTime(5000);
		await flushPromises();
		expect(getStatus).toHaveBeenCalledTimes(2);
		expect(vi.mocked(getStatus).mock.calls.every(([id]) => id === "printer")).toBe(true);

		FakeEventSource.last().open();
		await flushPromises();
		expect(getStatus).toHaveBeenCalledTimes(3);
		vi.advanceTimersByTime(10000);
		await flushPromises();
		expect(getStatus).toHaveBeenCalledTimes(3);
		wrapper.unmount();
	});
});
