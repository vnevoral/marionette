import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	ApiError,
	REQUEST_TIMEOUT_MS,
	connectStatusEvents,
	createCard,
	deleteCard,
	enqueuePrimary,
	enqueueStatus,
	getCard,
	getRuns,
	getStatus,
	getStatusHistory,
	listCards,
	updateCard,
	type ActionCard,
	type StatusEvent,
} from "@/api";

type FetchMock = ReturnType<typeof vi.fn<typeof fetch>>;

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { "Content-Type": "application/json" },
	});
}

const card: ActionCard = {
	id: "card-1",
	name: "Card",
	primary: { command: "true", timeoutSec: 5, rule: { type: "exit_code" } },
	currentStatus: {
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

describe("api request helper", () => {
	let fetchMock: FetchMock;

	beforeEach(() => {
		fetchMock = vi.fn<typeof fetch>();
		vi.stubGlobal("fetch", fetchMock);
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it("parses 2xx JSON bodies and sends the Accept header", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(200, [card]));
		const cards = await listCards();
		expect(cards).toHaveLength(1);
		expect(cards[0].id).toBe("card-1");
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe("/api/cards");
		expect(init?.headers).toMatchObject({ Accept: "application/json" });
	});

	it("resolves to undefined on 204 without reading a body", async () => {
		fetchMock.mockImplementation(async () => new Response(null, { status: 204 }));
		await expect(deleteCard("card 1")).resolves.toBeUndefined();
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe("/api/cards/card%201");
		expect(init?.method).toBe("DELETE");
	});

	it("keeps the Accept header when a call adds its own headers", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(201, card));
		await createCard(card);
		const [, init] = fetchMock.mock.calls[0];
		expect(init?.headers).toMatchObject({
			Accept: "application/json",
			"Content-Type": "application/json",
		});
		expect(init?.method).toBe("POST");
	});

	it("strips the runtime status from create and update payloads", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(201, card));
		await createCard(card);
		const [, init] = fetchMock.mock.calls[0];
		const payload = JSON.parse(String(init?.body)) as Record<string, unknown>;
		expect(payload).not.toHaveProperty("currentStatus");
		expect(payload.id).toBe("card-1");
	});

	it("declares application/json on mutating calls without a body (NFR-12)", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(202, { status: "accepted" }));
		await enqueuePrimary("card-1");
		await enqueueStatus("card-1");
		for (const [url, init] of fetchMock.mock.calls) {
			expect(String(url)).toMatch(/^\/api\/cards\/card-1\/actions\//);
			expect(init?.method).toBe("POST");
			expect(init?.headers).toMatchObject({ "Content-Type": "application/json" });
		}
	});

	it("throws an ApiError with status and fields for 4xx JSON envelopes", async () => {
		fetchMock.mockImplementation(async () =>
			jsonResponse(422, {
				error: "validation failed: name: required",
				fields: { name: "required", "primary.command": "required", ignored: 1 },
			}),
		);
		const failure = await getCard("card-1").catch((error: unknown) => error);
		expect(failure).toBeInstanceOf(ApiError);
		const apiError = failure as ApiError;
		expect(apiError.message).toBe("validation failed: name: required");
		expect(apiError.status).toBe(422);
		expect(apiError.fields).toEqual({ name: "required", "primary.command": "required" });
		expect(apiError.isNotFound).toBe(false);
	});

	it("flags 404 responses", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(404, { error: "card not found" }));
		const failure = (await getCard("missing").catch((error: unknown) => error)) as ApiError;
		expect(failure.isNotFound).toBe(true);
		expect(failure.fields).toBeUndefined();
	});

	it("rejects a 2xx response that is not JSON", async () => {
		fetchMock.mockImplementation(async () => new Response("<html>login</html>", { status: 200 }));
		await expect(listCards()).rejects.toThrow("Unexpected response from server");
	});

	it("maps network failures to Server unreachable with status 0", async () => {
		fetchMock.mockRejectedValue(new TypeError("Failed to fetch"));
		const failure = (await listCards().catch((error: unknown) => error)) as ApiError;
		expect(failure).toBeInstanceOf(ApiError);
		expect(failure.message).toBe("Server unreachable");
		expect(failure.isUnreachable).toBe(true);
	});

	it("aborts a request that exceeds the timeout budget", async () => {
		vi.useFakeTimers();
		try {
			fetchMock.mockImplementation(
				(_url, init) =>
					new Promise<Response>((_resolve, reject) => {
						init?.signal?.addEventListener("abort", () =>
							reject(new DOMException("aborted", "AbortError")),
						);
					}),
			);
			const pending = listCards();
			const outcome = pending.catch((error: unknown) => error);
			await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS);
			const failure = (await outcome) as ApiError;
			expect(failure).toBeInstanceOf(ApiError);
			expect(failure.message).toBe("Request timed out");
			expect(failure.status).toBe(0);
		} finally {
			vi.useRealTimers();
		}
	});

	it("falls back to the HTTP status when a 5xx body is not JSON", async () => {
		fetchMock.mockImplementation(
			async () => new Response("<html>bad gateway</html>", { status: 502 }),
		);
		await expect(listCards()).rejects.toThrow("Request failed (502)");
	});

	it("falls back to the HTTP status when the envelope has no error field", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(500, { message: "boom" }));
		await expect(listCards()).rejects.toThrow("Request failed (500)");
	});
});

describe("api read helpers", () => {
	let fetchMock: FetchMock;

	beforeEach(() => {
		fetchMock = vi.fn<typeof fetch>();
		vi.stubGlobal("fetch", fetchMock);
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it("builds card-scoped URLs with encoded identifiers", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(200, []));
		await getStatus("a/b");
		await getRuns("a/b");
		await getStatusHistory("a/b");
		await getCard("a/b");
		expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
			"/api/cards/a%2Fb/status",
			"/api/cards/a%2Fb/runs",
			"/api/cards/a%2Fb/status/history",
			"/api/cards/a%2Fb",
		]);
	});

	it("updates a card with PUT to its own URL", async () => {
		fetchMock.mockImplementation(async () => jsonResponse(200, card));
		await updateCard(card);
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe("/api/cards/card-1");
		expect(init?.method).toBe("PUT");
		expect(JSON.parse(String(init?.body))).not.toHaveProperty("currentStatus");
	});
});

describe("connectStatusEvents", () => {
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	class FakeEventSource {
		static instances: FakeEventSource[] = [];
		readonly url: string;
		private listeners = new Map<string, (event: Event) => void>();

		constructor(url: string) {
			this.url = url;
			FakeEventSource.instances.push(this);
		}

		addEventListener(type: string, listener: (event: Event) => void) {
			this.listeners.set(type, listener);
		}

		emit(type: string, event: Event) {
			this.listeners.get(type)?.(event);
		}
	}

	it("returns undefined when EventSource is unavailable", () => {
		vi.stubGlobal("EventSource", undefined);
		expect(connectStatusEvents(() => {})).toBeUndefined();
	});

	it("subscribes to status.changed and delivers parsed events only", () => {
		FakeEventSource.instances = [];
		vi.stubGlobal("EventSource", FakeEventSource);
		const received: StatusEvent[] = [];
		const source = connectStatusEvents((event) => received.push(event));
		expect(source).toBeDefined();
		const fake = FakeEventSource.instances[0];
		expect(fake.url).toBe("/api/events");

		const payload: StatusEvent = { cardId: "card-1", snapshot: card.currentStatus! };
		fake.emit(
			"status.changed",
			new MessageEvent("status.changed", { data: JSON.stringify(payload) }),
		);
		fake.emit("status.changed", new MessageEvent("status.changed", { data: "{not json" }));
		fake.emit("status.changed", new Event("status.changed"));
		expect(received).toEqual([payload]);
	});
});
