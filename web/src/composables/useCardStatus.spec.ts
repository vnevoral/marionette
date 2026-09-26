import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { effectScope, ref } from "vue";
import { getStatus, type ActionCard, type StatusSnapshot } from "@/api";
import {
	defaultMaxWaitMs,
	expectsFollowUpCheck,
	isNewerCheck,
	statusCheckQueueMarginMs,
	supersedes,
	useCardStatus,
	waitBudgetMs,
	waitForNewerStatus,
} from "@/composables/useCardStatus";
import { FakeEventSource } from "@/test/fakeEventSource";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return { ...actual, getStatus: vi.fn() };
});

const getStatusMock = vi.mocked(getStatus);

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

const before = "2026-09-26T10:00:00Z";
const after = "2026-09-26T10:00:05Z";

describe("isNewerCheck", () => {
	it("requires a real checkedAt that differs from the previous one", () => {
		expect(isNewerCheck(snapshot(after), before)).toBe(true);
		expect(isNewerCheck(snapshot(after), undefined)).toBe(true);
		expect(isNewerCheck(snapshot(before), before)).toBe(false);
		expect(isNewerCheck(snapshot(before), after)).toBe(false);
		expect(isNewerCheck({ state: "unknown" }, before)).toBe(false);
		expect(isNewerCheck({ state: "unknown" }, undefined)).toBe(false);
	});
});

describe("supersedes", () => {
	it("lets a newer or equal check replace the current one, never an older or unchecked one", () => {
		expect(supersedes(snapshot(after), snapshot(before))).toBe(true);
		expect(supersedes(snapshot(after), snapshot(after))).toBe(true);
		expect(supersedes(snapshot(before), snapshot(after))).toBe(false);
		expect(supersedes({ state: "unknown" }, snapshot(before))).toBe(false);
		expect(supersedes(snapshot(before), undefined)).toBe(true);
		expect(supersedes(snapshot(before), { state: "unknown" })).toBe(true);
		expect(supersedes({ state: "unknown" }, undefined)).toBe(true);
	});
});

describe("expectsFollowUpCheck", () => {
	const action = { command: "true", timeoutSec: 5, rule: { type: "exit_code" as const } };
	const polled: ActionCard = {
		id: "polled",
		name: "Polled",
		primary: action,
		status: action,
		pollingIntervalSeconds: 60,
		fastPollingIntervalSeconds: 10,
		fastPollingWindowSeconds: 120,
	};

	it("mirrors the scheduler: a primary action is followed by a check only with full polling", () => {
		expect(expectsFollowUpCheck(polled, "primary")).toBe(true);
		expect(expectsFollowUpCheck({ ...polled, pollingIntervalSeconds: 0 }, "primary")).toBe(false);
		expect(
			expectsFollowUpCheck({ ...polled, fastPollingIntervalSeconds: undefined }, "primary"),
		).toBe(false);
		expect(expectsFollowUpCheck({ ...polled, fastPollingWindowSeconds: 0 }, "primary")).toBe(false);
		expect(expectsFollowUpCheck({ ...polled, status: undefined }, "primary")).toBe(false);
	});

	it("always expects a check after a status action on a card that has one", () => {
		expect(expectsFollowUpCheck({ ...polled, pollingIntervalSeconds: 0 }, "status")).toBe(true);
		expect(expectsFollowUpCheck({ ...polled, status: undefined }, "status")).toBe(false);
	});

	it("waits for the fast polling window after a primary action, or the default budget", () => {
		expect(waitBudgetMs(polled, "primary")).toBe(120_000);
		expect(waitBudgetMs({ ...polled, fastPollingWindowSeconds: 30 }, "primary")).toBe(30_000);
		expect(waitBudgetMs({ ...polled, fastPollingWindowSeconds: undefined }, "primary")).toBe(
			defaultMaxWaitMs,
		);
	});

	it("bounds a manual status check by the status action timeout plus the queue margin", () => {
		const slow = { ...polled, fastPollingWindowSeconds: 10, status: { ...action, timeoutSec: 30 } };
		expect(waitBudgetMs(slow, "status")).toBe(30_000 + statusCheckQueueMarginMs);
		expect(waitBudgetMs({ ...slow, status: undefined }, "status")).toBe(10_000);
	});
});

describe("waitForNewerStatus", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
		vi.useFakeTimers();
		getStatusMock.mockReset();
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it("resolves with updated when a stream event carries a newer checkedAt", async () => {
		const seen: StatusSnapshot[] = [];
		const wait = waitForNewerStatus("card-1", before, {
			onSnapshot: (next) => seen.push(next),
		});
		const source = FakeEventSource.last();
		source.status({ cardId: "other", snapshot: snapshot(after) });
		source.status({ cardId: "card-1", snapshot: snapshot(before, "fail") });
		source.status({ cardId: "card-1", snapshot: snapshot(after) });
		await expect(wait).resolves.toBe("updated");
		expect(seen.map((item) => item.checkedAt)).toEqual([before, after]);
		expect(source.closed).toBe(true);
	});

	it("polls REST every interval regardless of the stream mode", async () => {
		getStatusMock.mockResolvedValueOnce(snapshot(before)).mockResolvedValueOnce(snapshot(after));
		const wait = waitForNewerStatus("card-1", before, { pollIntervalMs: 2000 });
		FakeEventSource.last().open();
		await vi.advanceTimersByTimeAsync(2000);
		expect(getStatusMock).toHaveBeenCalledTimes(1);
		await vi.advanceTimersByTimeAsync(2000);
		expect(getStatusMock).toHaveBeenCalledTimes(2);
		await expect(wait).resolves.toBe("updated");
		await vi.advanceTimersByTimeAsync(4000);
		expect(getStatusMock).toHaveBeenCalledTimes(2);
	});

	it("reports REST failures and keeps waiting", async () => {
		getStatusMock
			.mockRejectedValueOnce(new Error("offline"))
			.mockResolvedValueOnce(snapshot(after));
		const onError = vi.fn();
		const wait = waitForNewerStatus("card-1", before, { pollIntervalMs: 1000, onError });
		await vi.advanceTimersByTimeAsync(1000);
		expect(onError).toHaveBeenCalledTimes(1);
		await vi.advanceTimersByTimeAsync(1000);
		await expect(wait).resolves.toBe("updated");
	});

	it("resolves with timeout after maxWaitMs", async () => {
		getStatusMock.mockResolvedValue(snapshot(before));
		const wait = waitForNewerStatus("card-1", before, { maxWaitMs: 5000, pollIntervalMs: 1000 });
		await vi.advanceTimersByTimeAsync(5000);
		await expect(wait).resolves.toBe("timeout");
	});

	it("resolves with aborted and writes nothing afterwards", async () => {
		getStatusMock.mockResolvedValue(snapshot(after));
		const controller = new AbortController();
		const onSnapshot = vi.fn();
		const wait = waitForNewerStatus("card-1", before, {
			signal: controller.signal,
			pollIntervalMs: 1000,
			onSnapshot,
		});
		controller.abort();
		await expect(wait).resolves.toBe("aborted");
		FakeEventSource.last().status({ cardId: "card-1", snapshot: snapshot(after) });
		await vi.advanceTimersByTimeAsync(3000);
		expect(onSnapshot).not.toHaveBeenCalled();
		expect(getStatusMock).not.toHaveBeenCalled();

		const aborted = new AbortController();
		aborted.abort();
		await expect(waitForNewerStatus("card-1", before, { signal: aborted.signal })).resolves.toBe(
			"aborted",
		);
	});
});

describe("useCardStatus", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
		vi.useFakeTimers();
		getStatusMock.mockReset();
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it("follows stream events for its card only and releases the stream with its scope", () => {
		const scope = effectScope();
		const status = scope.run(() => useCardStatus("card-1"))!;
		const source = FakeEventSource.last();
		source.status({ cardId: "other", snapshot: snapshot(after) });
		expect(status.snapshot.value).toBeNull();
		source.status({ cardId: "card-1", snapshot: snapshot(after) });
		expect(status.snapshot.value?.checkedAt).toBe(after);
		scope.stop();
		expect(source.closed).toBe(true);
	});

	it("refreshes over REST on polling ticks only while enabled and ignores stale ids", async () => {
		const cardId = ref("card-1");
		const enabled = ref(false);
		const scope = effectScope();
		const status = scope.run(() => useCardStatus(cardId, enabled))!;
		getStatusMock.mockResolvedValue(snapshot(after));

		await vi.advanceTimersByTimeAsync(5000);
		expect(getStatusMock).not.toHaveBeenCalled();

		enabled.value = true;
		await vi.advanceTimersByTimeAsync(5000);
		expect(getStatusMock).toHaveBeenCalledWith("card-1");
		expect(status.snapshot.value?.checkedAt).toBe(after);

		let resolveLate!: (value: StatusSnapshot) => void;
		getStatusMock.mockImplementationOnce(
			() => new Promise<StatusSnapshot>((resolve) => (resolveLate = resolve)),
		);
		const pending = status.refresh();
		cardId.value = "card-2";
		status.set(undefined);
		resolveLate(snapshot(before));
		await pending;
		expect(status.snapshot.value).toBeNull();

		getStatusMock.mockRejectedValueOnce(new Error("offline"));
		await status.refresh();
		expect(status.failed.value).toBe(true);
		scope.stop();
	});

	it("never replaces a newer snapshot with an older one, from any source", () => {
		const scope = effectScope();
		const status = scope.run(() => useCardStatus("card-1"))!;
		FakeEventSource.last().status({ cardId: "card-1", snapshot: snapshot(after, "fail") });
		expect(status.snapshot.value?.state).toBe("fail");

		// A card loaded while the event was in flight carries the older check.
		status.set(snapshot(before));
		expect(status.snapshot.value?.checkedAt).toBe(after);
		// A slow REST poll answering late, and a poll for another card.
		status.apply("card-1", snapshot(before));
		status.apply("card-2", snapshot("2026-09-26T11:00:00Z"));
		expect(status.snapshot.value?.checkedAt).toBe(after);

		status.fail("card-2");
		expect(status.failed.value).toBe(false);
		status.fail("card-1");
		expect(status.failed.value).toBe(true);
		status.apply("card-1", snapshot(after));
		expect(status.failed.value).toBe(false);

		status.set(undefined);
		expect(status.snapshot.value).toBeNull();
		scope.stop();
	});
});
