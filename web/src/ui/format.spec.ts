import { describe, expect, it } from "vitest";
import {
	elapsedSince,
	formatDate,
	formatDuration,
	formatElapsed,
	lastCheckedLabel,
	transitionDuration,
} from "@/ui/format";

describe("format", () => {
	it("formats Go durations given in nanoseconds", () => {
		expect(formatDuration(250_000_000)).toBe("250 ms");
		expect(formatDuration(1_500_000_000)).toBe("1.50 s");
		expect(formatDuration(90_000_000_000)).toBe("1 min");
	});

	it("formats elapsed wall-clock spans coarsely", () => {
		expect(formatElapsed(30_000)).toBe("under a minute");
		expect(formatElapsed(5 * 60_000)).toBe("5 min");
		expect(formatElapsed(125 * 60_000)).toBe("2 h 05 min");
		expect(formatElapsed(26 * 3_600_000)).toBe("1 d 2 h");
		expect(formatElapsed(-5)).toBe("under a minute");
	});

	it("labels the last check or its absence", () => {
		expect(lastCheckedLabel(undefined)).toBe("Not checked yet");
		expect(lastCheckedLabel("2026-09-26T10:00:00Z")).toMatch(/^Last checked /);
		expect(formatDate(undefined)).toBe("Not available");
	});

	it("uses the stored duration for ended transitions and elapsed time for the current one", () => {
		const now = Date.parse("2026-09-26T12:00:00Z");
		expect(
			transitionDuration(
				{
					state: "ok",
					startedAt: "2026-09-26T10:00:00Z",
					endedAt: "2026-09-26T10:00:03Z",
					duration: 3_000_000_000,
				},
				now,
			),
		).toBe("3.00 s");
		expect(
			transitionDuration({ state: "fail", startedAt: "2026-09-26T11:15:00Z", duration: 0 }, now),
		).toBe("45 min");
		expect(elapsedSince("2026-09-26T11:59:30Z", now)).toBe("under a minute");
	});
});
