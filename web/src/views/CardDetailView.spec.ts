import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { createMemoryHistory, createRouter } from "vue-router";
import PrimeVue from "primevue/config";
import {
	enqueuePrimary,
	getCard,
	getRuns,
	getStatusHistory,
	type ActionCard,
	type StatusSnapshot,
} from "@/api";
import CardDetailView from "@/views/CardDetailView.vue";
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
		currentStatus: snapshot(before),
	},
	lamp: {
		id: "lamp",
		name: "Lamp",
		primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	},
};

function buttonByLabel(wrapper: ReturnType<typeof mount>, label: string) {
	const button = wrapper.findAll("button").find((candidate) => candidate.text().includes(label));
	if (!button) throw new Error(`button ${label} not rendered`);
	return button;
}

async function mountDetail(id: string) {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{ path: "/", component: { template: "<div />" } },
			{ path: "/cards/:id", name: "card-detail", component: CardDetailView },
		],
	});
	await router.push(`/cards/${id}`);
	await router.isReady();
	const wrapper = mount(CardDetailView, { global: { plugins: [router, PrimeVue] } });
	await flushPromises();
	return { wrapper, router };
}

describe("CardDetailView", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
		vi.mocked(getCard)
			.mockReset()
			.mockImplementation(async (id) => {
				const card = cards[id];
				if (!card) throw new Error("card not found");
				return card;
			});
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
		expect(wrapper.find(".request-feedback").text()).toBe("Action queued");
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeDefined();
		expect(buttonByLabel(wrapper, "Check status").attributes("disabled")).toBeDefined();
		expect(wrapper.text()).toContain("Running");

		FakeEventSource.last().status({ cardId: "printer", snapshot: snapshot(after, "fail") });
		await flushPromises();
		expect(wrapper.find(".request-feedback").text()).toBe("Status updated");
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeUndefined();
		expect(buttonByLabel(wrapper, "Check status").attributes("disabled")).toBeUndefined();
		expect(wrapper.text()).toContain("Problem");
		expect(getRuns).toHaveBeenCalledTimes(2);
		expect(getStatusHistory).toHaveBeenCalledTimes(2);
		wrapper.unmount();
		expect(FakeEventSource.last().closed).toBe(true);
	});

	it("shows the enqueue error and re-enables the buttons", async () => {
		vi.mocked(enqueuePrimary).mockRejectedValue(new Error("queue is full"));
		const { wrapper } = await mountDetail("printer");
		await buttonByLabel(wrapper, "Run action").trigger("click");
		await flushPromises();
		expect(wrapper.find(".request-feedback").text()).toBe("queue is full");
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
		expect(wrapper.find(".request-feedback").text()).toBe("Action accepted");
		expect(buttonByLabel(wrapper, "Run action").attributes("disabled")).toBeUndefined();
		expect(getRuns).toHaveBeenCalledTimes(2);
		wrapper.unmount();
	});

	it("shows the not-found state when the card cannot be loaded", async () => {
		const { wrapper } = await mountDetail("missing");
		expect(wrapper.text()).toContain("Card unavailable");
		expect(wrapper.text()).toContain("card not found");
		wrapper.unmount();
	});
});
