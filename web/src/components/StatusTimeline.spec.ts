import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import type { StatusChange } from "@/api";
import StatusTimeline from "@/components/StatusTimeline.vue";

const now = Date.parse("2026-09-26T12:00:00Z");
const changes: StatusChange[] = [
	{ state: "fail", startedAt: "2026-09-26T11:30:00Z", duration: 0 },
	{
		state: "ok",
		startedAt: "2026-09-26T09:30:00Z",
		endedAt: "2026-09-26T11:30:00Z",
		duration: 2 * 3_600_000_000_000,
	},
];

function mountTimeline(
	props: Partial<{ changes: StatusChange[]; loading: boolean; error: string }>,
) {
	return mount(StatusTimeline, {
		props: { changes: [], loading: false, now, ...props },
		global: { plugins: [PrimeVue] },
	});
}

describe("StatusTimeline", () => {
	it("shows the elapsed time for the current transition and the stored duration for ended ones", () => {
		const wrapper = mountTimeline({ changes });
		const rows = wrapper.findAll(".timeline-row");
		expect(rows).toHaveLength(2);
		expect(rows[0].find(".status-badge").text()).toBe("Problem");
		expect(rows[0].text()).toContain("30 min");
		expect(rows[0].text()).toContain("(current)");
		expect(rows[1].find(".status-badge").text()).toBe("Healthy");
		expect(rows[1].text()).toContain("2 h 00 min");
		expect(rows[1].text()).not.toContain("(current)");
	});

	it("renders loading, empty and error states", () => {
		expect(mountTimeline({ loading: true }).text()).toBe("Loading status history...");
		expect(mountTimeline({}).text()).toBe("No status transitions recorded yet.");
		expect(
			mountTimeline({ error: "Status history is unavailable" }).find('[role="alert"]').text(),
		).toBe("Status history is unavailable");
	});
});
