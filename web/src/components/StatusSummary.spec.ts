import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import type { Run, StatusSnapshot } from "@/api";
import StatusSummary from "@/components/StatusSummary.vue";

function check(overrides: Partial<Run> = {}): Run {
	return {
		actionKind: "status",
		startedAt: "2026-09-26T10:00:00Z",
		duration: 45_000_000,
		exitCode: 2,
		output: "/usr/bin/ping: socket: Operation not permitted\n",
		truncated: false,
		outcome: "fail",
		...overrides,
	};
}

function snapshot(lastCheck?: Run): StatusSnapshot {
	return { state: "fail", checkedAt: "2026-09-26T10:00:00Z", lastCheck };
}

function mountSummary(props: {
	hasStatus?: boolean;
	status?: StatusSnapshot | null;
	unavailable?: boolean;
}) {
	return mount(StatusSummary, {
		props: { hasStatus: true, ...props },
		global: { plugins: [PrimeVue] },
	});
}

function row(wrapper: ReturnType<typeof mountSummary>, term: string) {
	return wrapper
		.findAll(".summary-list div")
		.find((item) => item.find("dt").text() === term)
		?.find("dd")
		.text();
}

describe("StatusSummary", () => {
	it("shows the exit code, duration and output of the last check on demand (FR-21a)", () => {
		const wrapper = mountSummary({ status: snapshot(check()) });
		expect(row(wrapper, "Last outcome")).toBe("Failed");
		expect(row(wrapper, "Exit code")).toBe("2");
		expect(row(wrapper, "Duration")).toBeTruthy();
		const output = wrapper.find("details.run-output");
		expect(output.exists()).toBe(true);
		expect(output.attributes("open")).toBeUndefined();
		expect(output.find("summary").text()).toBe("View output");
		expect(output.find("pre").text()).toContain("socket: Operation not permitted");
		expect(output.find("pre").text()).not.toContain("...");
	});

	it("marks truncated output", () => {
		const wrapper = mountSummary({ status: snapshot(check({ truncated: true })) });
		expect(wrapper.find("details.run-output pre").text()).toMatch(/\.\.\.$/);
	});

	it("renders no output block for an empty output", () => {
		const wrapper = mountSummary({ status: snapshot(check({ output: "", exitCode: 0 })) });
		expect(row(wrapper, "Exit code")).toBe("0");
		expect(wrapper.find("details").exists()).toBe(false);
	});

	it("shows nothing extra before the first check", () => {
		const wrapper = mountSummary({ status: snapshot(undefined) });
		expect(row(wrapper, "Last outcome")).toBe("Not checked");
		expect(row(wrapper, "Exit code")).toBeUndefined();
		expect(wrapper.find("details").exists()).toBe(false);
	});

	it("keeps the unavailable note alongside the last check (block 0040)", () => {
		const wrapper = mountSummary({ status: snapshot(check()), unavailable: true });
		expect(wrapper.find(".summary-unavailable").text()).toBe("Status could not be refreshed");
		expect(wrapper.find("details.run-output").exists()).toBe(true);
	});

	it("explains a card without a status action", () => {
		const wrapper = mountSummary({ hasStatus: false, status: null });
		expect(wrapper.find(".summary-list").exists()).toBe(false);
		expect(wrapper.find("details").exists()).toBe(false);
		expect(wrapper.text()).toContain("Status monitoring is not configured");
	});
});
