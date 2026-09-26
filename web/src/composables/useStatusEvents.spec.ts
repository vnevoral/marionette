import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { StatusEvent } from "@/api";
import { createStatusEvents, type StatusListener } from "@/composables/useStatusEvents";
import { FakeEventSource } from "@/test/fakeEventSource";

const event: StatusEvent = {
	cardId: "card-1",
	snapshot: {
		state: "ok",
		checkedAt: "2026-09-26T10:00:00Z",
		lastCheck: {
			actionKind: "status",
			startedAt: "2026-09-26T10:00:00Z",
			duration: 1,
			exitCode: 0,
			output: "",
			truncated: false,
			outcome: "ok",
		},
	},
};

function listener() {
	return {
		onStatus: vi.fn<(event: StatusEvent) => void>(),
		onRefresh: vi.fn<() => void>(),
	} satisfies StatusListener;
}

describe("createStatusEvents", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	function connector() {
		return (onStatus: (event: StatusEvent) => void) => {
			const source = new FakeEventSource("/api/events");
			source.addEventListener("status.changed", (raw) => {
				if (raw instanceof MessageEvent) onStatus(JSON.parse(raw.data) as StatusEvent);
			});
			return source as unknown as EventSource;
		};
	}

	it("shares one connection between subscribers and closes after the last one leaves", () => {
		const events = createStatusEvents(connector(), 1000);
		const first = listener();
		const second = listener();
		const leaveFirst = events.subscribe(first);
		const leaveSecond = events.subscribe(second);
		expect(FakeEventSource.instances).toHaveLength(1);
		expect(events.subscribers).toBe(2);

		FakeEventSource.last().status(event);
		expect(first.onStatus).toHaveBeenCalledWith(event);
		expect(second.onStatus).toHaveBeenCalledWith(event);

		leaveFirst();
		leaveFirst();
		expect(FakeEventSource.last().closed).toBe(false);
		leaveSecond();
		expect(FakeEventSource.last().closed).toBe(true);
		expect(events.subscribers).toBe(0);

		events.subscribe(listener());
		expect(FakeEventSource.instances).toHaveLength(2);
	});

	it("polls until the stream opens, refreshes once on open and polls again after an error", () => {
		const events = createStatusEvents(connector(), 1000);
		const subscriber = listener();
		events.subscribe(subscriber);
		expect(events.connected.value).toBe(false);
		expect(events.mode.value).toBe("polling");

		vi.advanceTimersByTime(2000);
		expect(subscriber.onRefresh).toHaveBeenCalledTimes(2);

		FakeEventSource.last().open();
		expect(events.connected.value).toBe(true);
		expect(events.mode.value).toBe("live");
		expect(subscriber.onRefresh).toHaveBeenCalledTimes(3);
		vi.advanceTimersByTime(5000);
		expect(subscriber.onRefresh).toHaveBeenCalledTimes(3);

		FakeEventSource.last().fail();
		expect(events.connected.value).toBe(false);
		expect(events.mode.value).toBe("polling");
		vi.advanceTimersByTime(1000);
		expect(subscriber.onRefresh).toHaveBeenCalledTimes(4);
	});

	it("falls back to polling when EventSource is unavailable", () => {
		const events = createStatusEvents(() => undefined, 1000);
		const subscriber = listener();
		const leave = events.subscribe(subscriber);
		vi.advanceTimersByTime(3000);
		expect(subscriber.onRefresh).toHaveBeenCalledTimes(3);
		expect(events.connected.value).toBe(false);
		leave();
		vi.advanceTimersByTime(3000);
		expect(subscriber.onRefresh).toHaveBeenCalledTimes(3);
	});
});
