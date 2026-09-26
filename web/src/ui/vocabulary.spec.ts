import { describe, expect, it } from "vitest";
import type { Run, StatusState } from "@/api";
import {
	NO_STATUS_CHECK,
	OUTCOME,
	REQUEST,
	STATUS,
	outcomePresentation,
	statusPresentation,
} from "@/ui/vocabulary";

const states: StatusState[] = ["unknown", "ok", "fail"];
const outcomes: Run["outcome"][] = ["ok", "fail", "timeout", "canceled"];
const rawValues = ["ok", "fail", "accepted", "running", "timeout"];

describe("vocabulary", () => {
	it("presents every device state with label, icon and tone and never a raw value", () => {
		for (const state of states) {
			const presentation = statusPresentation(state);
			expect(presentation).toBe(STATUS[state]);
			expect(presentation.label).toBeTruthy();
			expect(presentation.icon).toMatch(/^pi /);
			expect(rawValues).not.toContain(presentation.label.toLowerCase());
		}
		expect(statusPresentation("ok").label).toBe("Healthy");
		expect(statusPresentation("fail").label).toBe("Problem");
		expect(statusPresentation(undefined)).toBe(STATUS.unknown);
		expect(statusPresentation("bogus")).toBe(STATUS.unknown);
	});

	it("presents every run outcome", () => {
		for (const outcome of outcomes) {
			expect(outcomePresentation(outcome)).toBe(OUTCOME[outcome]);
			expect(rawValues).not.toContain(OUTCOME[outcome].label.toLowerCase());
		}
		expect(outcomePresentation("nonsense")).toBe(OUTCOME.fail);
	});

	it("distinguishes queued (amber) from running (blue) as the spec requires", () => {
		expect(REQUEST.queued.tone).toBe("warning");
		expect(REQUEST.running.tone).toBe("info");
		expect(REQUEST.queued.label).toBe("Queued");
		expect(REQUEST.running.label).toBe("Running");
		expect(NO_STATUS_CHECK.tone).toBe("unknown");
	});
});
