export type StatusState = "unknown" | "ok" | "fail";

export interface Action {
	command: string;
	args?: string[];
	dir?: string;
	env?: Record<string, string>;
	timeoutSec: number;
	rule: { type: "exit_code" | "match" | "not_match"; pattern?: string };
}

export interface ActionCard {
	id: string;
	name: string;
	description?: string;
	icon?: string;
	primary: Action;
	status?: Action;
	pollingIntervalSeconds?: number;
	fastPollingIntervalSeconds?: number;
	fastPollingWindowSeconds?: number;
	currentStatus?: StatusSnapshot;
}

export interface Run {
	actionKind: string;
	startedAt: string;
	/** Nanoseconds (Go `time.Duration`); see the API contract in the architecture overview. */
	duration: number;
	exitCode: number;
	output: string;
	truncated: boolean;
	outcome: "ok" | "fail" | "timeout" | "canceled";
}

export interface StatusSnapshot {
	state: StatusState;
	/** Absent until the card has been checked at least once. */
	checkedAt?: string;
	/** Absent until the card has been checked at least once. */
	lastCheck?: Run;
}

export interface StatusEvent {
	cardId: string;
	snapshot: StatusSnapshot;
}

/** 202 response of the enqueue endpoints. */
export interface AcceptedAction {
	cardId: string;
	actionKind: "primary" | "status";
	status: "accepted";
}

/** Request budget; a slow or hung server surfaces as a timeout, not a spinner. */
export const REQUEST_TIMEOUT_MS = 15_000;

/**
 * Error thrown by every API call. `status` is the HTTP status, or 0 when no
 * response arrived (network failure, timeout). `fields` carries the server's
 * per-field validation messages from a 422 response.
 */
export class ApiError extends Error {
	readonly status: number;
	readonly fields?: Record<string, string>;

	constructor(message: string, status: number, fields?: Record<string, string>) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.fields = fields;
	}

	get isNotFound() {
		return this.status === 404;
	}

	get isUnreachable() {
		return this.status === 0;
	}
}

interface ErrorEnvelope {
	error?: unknown;
	fields?: unknown;
}

function isJSON(response: Response) {
	return (response.headers.get("content-type") ?? "").toLowerCase().includes("application/json");
}

function fieldMessages(value: unknown): Record<string, string> | undefined {
	if (!value || typeof value !== "object") return undefined;
	const fields: Record<string, string> = {};
	for (const [key, message] of Object.entries(value)) {
		if (typeof message === "string") fields[key] = message;
	}
	return Object.keys(fields).length ? fields : undefined;
}

async function errorFromResponse(response: Response): Promise<ApiError> {
	const fallback = `Request failed (${response.status})`;
	if (!isJSON(response)) return new ApiError(fallback, response.status);
	try {
		const envelope = (await response.json()) as ErrorEnvelope;
		const message = typeof envelope.error === "string" ? envelope.error : fallback;
		return new ApiError(message, response.status, fieldMessages(envelope.fields));
	} catch {
		return new ApiError(fallback, response.status);
	}
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
	const { headers, signal, ...rest } = options;
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort("timeout"), REQUEST_TIMEOUT_MS);
	const forwardAbort = () => controller.abort(signal?.reason);
	if (signal?.aborted) forwardAbort();
	else signal?.addEventListener("abort", forwardAbort, { once: true });
	let response: Response;
	try {
		response = await fetch(path, {
			...rest,
			headers: { Accept: "application/json", ...headers },
			signal: controller.signal,
		});
	} catch (failure) {
		if (controller.signal.aborted && controller.signal.reason === "timeout") {
			throw new ApiError("Request timed out", 0);
		}
		if (failure instanceof DOMException && failure.name === "AbortError") throw failure;
		throw new ApiError("Server unreachable", 0);
	} finally {
		clearTimeout(timer);
		signal?.removeEventListener("abort", forwardAbort);
	}
	if (!response.ok) throw await errorFromResponse(response);
	if (response.status === 204) return undefined as T;
	if (!isJSON(response)) throw new ApiError("Unexpected response from server", response.status);
	try {
		return (await response.json()) as T;
	} catch {
		throw new ApiError("Unexpected response from server", response.status);
	}
}

function withoutRuntimeStatus(card: ActionCard): Omit<ActionCard, "currentStatus"> {
	const configuration = { ...card };
	delete configuration.currentStatus;
	return configuration;
}

export function listCards(): Promise<ActionCard[]> {
	return request<ActionCard[]>("/api/cards");
}

export function getCard(cardID: string): Promise<ActionCard> {
	return request<ActionCard>(`/api/cards/${encodeURIComponent(cardID)}`);
}

export function createCard(card: ActionCard): Promise<ActionCard> {
	return request<ActionCard>("/api/cards", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(withoutRuntimeStatus(card)),
	});
}

export function updateCard(card: ActionCard): Promise<ActionCard> {
	return request<ActionCard>(`/api/cards/${encodeURIComponent(card.id)}`, {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(withoutRuntimeStatus(card)),
	});
}

export function deleteCard(cardID: string): Promise<void> {
	return request(`/api/cards/${encodeURIComponent(cardID)}`, {
		method: "DELETE",
		headers: { "Content-Type": "application/json" },
	});
}

export function getStatus(cardID: string): Promise<StatusSnapshot> {
	return request<StatusSnapshot>(`/api/cards/${encodeURIComponent(cardID)}/status`);
}

export function getRuns(cardID: string): Promise<Run[]> {
	return request<Run[]>(`/api/cards/${encodeURIComponent(cardID)}/runs`);
}

export interface StatusChange {
	state: StatusState;
	startedAt: string;
	endedAt?: string;
	/** Nanoseconds (Go `time.Duration`). */
	duration: number;
}

export function getStatusHistory(cardID: string): Promise<StatusChange[]> {
	return request<StatusChange[]>(`/api/cards/${encodeURIComponent(cardID)}/status/history`);
}

export function connectStatusEvents(
	onStatusChange: (event: StatusEvent) => void,
): EventSource | undefined {
	if (typeof EventSource === "undefined") return undefined;
	const source = new EventSource("/api/events");
	source.addEventListener("status.changed", (event) => {
		if (!(event instanceof MessageEvent)) return;
		try {
			onStatusChange(JSON.parse(event.data) as StatusEvent);
		} catch {
			// Ignore malformed transient events; REST polling remains available.
		}
	});
	return source;
}

// Mutating calls always declare application/json, even without a body, so
// they pass the server's cross-site protection (NFR-12).
export function enqueuePrimary(cardID: string): Promise<AcceptedAction> {
	return request<AcceptedAction>(`/api/cards/${encodeURIComponent(cardID)}/actions/primary`, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
	});
}

export function enqueueStatus(cardID: string): Promise<AcceptedAction> {
	return request<AcceptedAction>(`/api/cards/${encodeURIComponent(cardID)}/actions/status/check`, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
	});
}
