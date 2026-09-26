import { describe, expect, it } from "vitest";
import { mount, RouterLinkStub } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import type { ActionCard as Card, StatusSnapshot } from "@/api";
import ActionCard from "@/components/ActionCard.vue";
import type { PendingRequest, RequestResult } from "@/types";

const card: Card = {
	id: "printer",
	name: "Printer",
	description: "Office printer",
	primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	status: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
};

function snapshot(state: StatusSnapshot["state"]): StatusSnapshot {
	return { state, checkedAt: "2026-09-26T10:00:00Z" };
}

function mountCard(props: {
	card?: Card;
	status?: StatusSnapshot | null;
	pending?: PendingRequest | null;
	result?: RequestResult | null;
}) {
	return mount(ActionCard, {
		props: { card, ...props },
		global: { plugins: [PrimeVue], stubs: { RouterLink: RouterLinkStub } },
	});
}

describe("ActionCard", () => {
	it.each([
		["ok", "Healthy", "status-healthy"],
		["fail", "Problem", "status-problem"],
		["unknown", "Unknown", "status-unknown"],
	] as const)("renders the %s state as %s", (state, label, className) => {
		const wrapper = mountCard({ status: snapshot(state) });
		const badge = wrapper.find(".status-badge");
		expect(badge.text()).toBe(label);
		expect(badge.classes()).toContain(className);
		expect(wrapper.text()).toContain("Last checked");
		expect(wrapper.findAll("button")).toHaveLength(2);
	});

	it("shows Not checked yet and the no-status badge appropriately", () => {
		expect(mountCard({ status: null }).text()).toContain("Not checked yet");
		const plain = mountCard({ card: { ...card, status: undefined } });
		expect(plain.find(".status-badge").text()).toBe("No status check");
		expect(plain.text()).not.toContain("Not checked yet");
		expect(plain.findAll("button")).toHaveLength(1);
	});

	it("shows queued in amber and running in blue while a request is pending", () => {
		const queued = mountCard({
			status: snapshot("ok"),
			pending: { action: "primary", phase: "queued" },
		});
		expect(queued.find(".status-badge").text()).toBe("Queued");
		expect(queued.find(".status-badge").classes()).toContain("status-warning");
		expect(queued.find(".request-feedback").text()).toBe("Queued");
		for (const button of queued.findAll("button")) {
			expect(button.attributes("disabled")).toBeDefined();
		}
		const running = mountCard({
			status: snapshot("ok"),
			pending: { action: "status", phase: "running" },
		});
		expect(running.find(".status-badge").classes()).toContain("status-info");
		expect(running.findAll("button")[1].classes()).toContain("p-button-loading");
	});

	it("shows the last result and emits run and check", async () => {
		const wrapper = mountCard({
			status: snapshot("ok"),
			result: { tone: "success", message: "Updated" },
		});
		expect(wrapper.find(".request-feedback").text()).toBe("Updated");
		expect(wrapper.find(".request-feedback").classes()).toContain("request-feedback-success");
		const [run, check] = wrapper.findAll("button");
		await run.trigger("click");
		await check.trigger("click");
		expect(wrapper.emitted("run")).toHaveLength(1);
		expect(wrapper.emitted("check")).toHaveLength(1);
		expect(wrapper.findComponent(RouterLinkStub).props("to")).toBe("/cards/printer");
	});
});
