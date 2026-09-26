import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	enqueuePrimary,
	enqueueStatus,
	getStatus,
	type ActionCard,
	type StatusSnapshot,
} from "@/api";
import { outcomeResult, requestAction } from "@/composables/useActionRequest";
import { FakeEventSource } from "@/test/fakeEventSource";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return { ...actual, getStatus: vi.fn(), enqueuePrimary: vi.fn(), enqueueStatus: vi.fn() };
});

const before = "2026-09-26T10:00:00Z";
const during = "2026-09-26T10:00:02Z";
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

const action = { command: "true", timeoutSec: 5, rule: { type: "exit_code" as const } };
const polled: ActionCard = {
	id: "printer",
	name: "Printer",
	primary: action,
	status: action,
	pollingIntervalSeconds: 60,
	fastPollingIntervalSeconds: 10,
	fastPollingWindowSeconds: 120,
};

function accepted(actionKind: "primary" | "status", checkedAt?: string) {
	return { cardId: "printer", actionKind, status: "accepted" as const, checkedAt };
}

describe("requestAction", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
		vi.useFakeTimers();
		vi.mocked(getStatus).mockReset();
		vi.mocked(enqueuePrimary).mockReset();
		vi.mocked(enqueueStatus).mockReset();
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it("reports acceptance without waiting when no check follows", async () => {
		vi.mocked(enqueuePrimary).mockResolvedValue(accepted("primary", before));
		const phases: string[] = [];
		const outcome = await requestAction({ ...polled, pollingIntervalSeconds: 0 }, "primary", {
			signal: new AbortController().signal,
			onPhase: (phase) => phases.push(phase),
		});
		expect(outcome).toEqual({ kind: "accepted" });
		expect(phases).toEqual(["queued"]);
		expect(getStatus).not.toHaveBeenCalled();
	});

	it("uses the check the server knew at acceptance as the baseline, not the one before the request", async () => {
		// A scheduled check completed while the request was in flight; the
		// server reports it in the 202 and a stream event for it arrives later.
		vi.mocked(enqueuePrimary).mockResolvedValue(accepted("primary", during));
		const phases: string[] = [];
		const seen: string[] = [];
		const pending = requestAction(polled, "primary", {
			signal: new AbortController().signal,
			onPhase: (phase) => phases.push(phase),
			onSnapshot: (next) => seen.push(next.checkedAt ?? ""),
		});
		await vi.advanceTimersByTimeAsync(0);
		expect(phases).toEqual(["queued", "running"]);
		const source = FakeEventSource.last();
		source.status({ cardId: "printer", snapshot: snapshot(before) });
		source.status({ cardId: "printer", snapshot: snapshot(during, "fail") });
		await vi.advanceTimersByTimeAsync(0);
		source.status({ cardId: "printer", snapshot: snapshot(after) });
		await expect(pending).resolves.toEqual({ kind: "updated" });
		expect(seen).toEqual([before, during, after]);
	});

	it("bounds a manual status check by its timeout and reports a timeout", async () => {
		vi.mocked(enqueueStatus).mockResolvedValue(accepted("status", before));
		vi.mocked(getStatus).mockResolvedValue(snapshot(before));
		const onError = vi.fn();
		const pending = requestAction(polled, "status", {
			signal: new AbortController().signal,
			onPhase: () => {},
			onError,
		});
		await vi.advanceTimersByTimeAsync(5_000 + 30_000);
		await expect(pending).resolves.toEqual({ kind: "timeout" });
		expect(onError).not.toHaveBeenCalled();
	});

	it("reports an enqueue failure with its message and an abort as aborted", async () => {
		vi.mocked(enqueuePrimary).mockRejectedValue(new Error("queue is full"));
		const failed = await requestAction(polled, "primary", {
			signal: new AbortController().signal,
			onPhase: () => {},
		});
		expect(failed).toEqual({ kind: "failed", message: "queue is full" });

		vi.mocked(enqueuePrimary).mockRejectedValue("boom");
		const unknown = await requestAction(polled, "primary", {
			signal: new AbortController().signal,
			onPhase: () => {},
		});
		expect(unknown).toEqual({ kind: "failed", message: "Unable to queue action" });

		vi.mocked(enqueuePrimary).mockResolvedValue(accepted("primary", before));
		const controller = new AbortController();
		const pending = requestAction(polled, "primary", {
			signal: controller.signal,
			onPhase: () => {},
		});
		await vi.advanceTimersByTimeAsync(0);
		controller.abort();
		await expect(pending).resolves.toEqual({ kind: "aborted" });
	});
});

describe("outcomeResult", () => {
	const labels = { accepted: "Accepted", updated: "Updated" };

	it("maps outcomes to feedback lines and hides an abort", () => {
		expect(outcomeResult({ kind: "accepted" }, labels)).toEqual({
			tone: "success",
			message: "Accepted",
		});
		expect(outcomeResult({ kind: "updated" }, labels)).toEqual({
			tone: "success",
			message: "Updated",
		});
		expect(outcomeResult({ kind: "timeout" }, labels)).toEqual({
			tone: "error",
			message: "Result not available yet",
		});
		expect(outcomeResult({ kind: "failed", message: "nope" }, labels)).toEqual({
			tone: "error",
			message: "nope",
		});
		expect(outcomeResult({ kind: "aborted" }, labels)).toBeNull();
	});
});
