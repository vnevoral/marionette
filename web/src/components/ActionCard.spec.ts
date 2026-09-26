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
	statusUnavailable?: boolean;
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

	it("marks the state as Unknown when the status could not be refreshed", () => {
		const stale = mountCard({ status: snapshot("ok"), statusUnavailable: true });
		expect(stale.find(".status-badge").text()).toBe("Unknown");
		expect(stale.find(".card-unavailable").text()).toBe("Status could not be refreshed");
		expect(stale.text()).toContain("Last checked");

		const running = mountCard({
			status: snapshot("ok"),
			statusUnavailable: true,
			pending: { action: "primary", phase: "running" },
		});
		expect(running.find(".status-badge").text()).toBe("Running");

		const plain = mountCard({ card: { ...card, status: undefined }, statusUnavailable: true });
		expect(plain.find(".card-unavailable").exists()).toBe(false);
	});

	it("shows queued in amber and running in blue while a request is pending", () => {
		const queued = mountCard({
			status: snapshot("ok"),
			pending: { action: "primary", phase: "queued" },
		});
		expect(queued.find(".status-badge").text()).toBe("Queued");
		expect(queued.find(".status-badge").classes()).toContain("status-warning");
		// No second "Queued" next to the badge: the line keeps Last checked
		// and the phase is only announced (UX spec §4).
		expect(queued.find(".action-note-text").text()).toMatch(/^Last checked /);
		expect(queued.find(".action-note [role='status']").text()).toBe("Queued");
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

	it("keeps an updated status quiet and shows other outcomes in place of Last checked", () => {
		const updated = mountCard({
			status: snapshot("ok"),
			result: { tone: "success", message: "Updated", quiet: true },
		});
		expect(updated.find(".action-note-text").text()).toMatch(/^Last checked /);
		expect(updated.find(".action-note [role='status']").text()).toBe("Updated");

		const failed = mountCard({
			status: snapshot("ok"),
			result: { tone: "error", message: "Result not available yet" },
		});
		expect(failed.find(".action-note-text").text()).toBe("Result not available yet");
		expect(failed.find(".action-note").classes()).toContain("action-note-error");
		expect(failed.find(".action-note").attributes("title")).toBe("Result not available yet");
		expect(failed.text()).not.toContain("Last checked");
		expect(failed.find(".request-feedback").exists()).toBe(false);
	});

	it("reserves the note line on a card without a status action", () => {
		const plain = mountCard({ card: { ...card, status: undefined } });
		expect(plain.find(".action-note").exists()).toBe(true);
		expect(plain.find(".action-note-text").text()).toBe("");

		const accepted = mountCard({
			card: { ...card, status: undefined },
			result: { tone: "success", message: "Accepted" },
		});
		expect(accepted.find(".action-note-text").text()).toBe("Accepted");
		expect(accepted.find(".action-note").classes()).toContain("action-note-success");
	});

	it("emits run and check and links to the detail", async () => {
		const wrapper = mountCard({ status: snapshot("ok") });
		const [run, check] = wrapper.findAll("button");
		await run.trigger("click");
		await check.trigger("click");
		expect(wrapper.emitted("run")).toHaveLength(1);
		expect(wrapper.emitted("check")).toHaveLength(1);
		expect(wrapper.findComponent(RouterLinkStub).props("to")).toBe("/cards/printer");
	});
});
