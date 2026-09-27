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
	/** A name from ui/cardColors.ts; empty or missing means no colour. */
	color?: string;
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

/** `run.recorded`: a primary run was written to the card's history (FR-42a). */
export interface RunEvent {
	cardId: string;
	run: Run;
}

/** 202 response of the enqueue endpoints. */
export interface AcceptedAction {
	cardId: string;
	actionKind: "primary" | "status";
	status: "accepted";
	/** Last check known to the server when it accepted the request; the
	 * baseline for "a check newer than the one before my action". Absent for
	 * a card that was never checked. */
	checkedAt?: string;
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
	/** The whole JSON error envelope, for endpoints that add their own keys. */
	readonly envelope?: Record<string, unknown>;

	constructor(
		message: string,
		status: number,
		fields?: Record<string, string>,
		envelope?: Record<string, unknown>,
	) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.fields = fields;
		this.envelope = envelope;
	}

	/** The device is not paired (FR-50). */
	get isUnauthorized() {
		return this.status === 401;
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
		return new ApiError(
			message,
			response.status,
			fieldMessages(envelope.fields),
			envelope && typeof envelope === "object" ? (envelope as Record<string, unknown>) : undefined,
		);
	} catch {
		return new ApiError(fallback, response.status);
	}
}

// Called when any API request answers 401 because this device is not (or no
// longer) paired; the router sends the user to the pairing screen. The
// session and pairing endpoints answer 401 as part of their contract and are
// excluded.
let unauthorizedHandler: (() => void) | undefined;
const unauthorizedIsExpected = new Set(["/api/session", "/api/pairing"]);

export function setUnauthorizedHandler(handler: (() => void) | undefined) {
	unauthorizedHandler = handler;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
	const { headers, signal, ...rest } = options;
	const controller = new AbortController();
	const timer = setTimeout(() => controller.abort("timeout"), REQUEST_TIMEOUT_MS);
	const forwardAbort = () => controller.abort(signal?.reason);
	if (signal?.aborted) forwardAbort();
	else signal?.addEventListener("abort", forwardAbort, { once: true });
	const timedOut = () => controller.signal.aborted && controller.signal.reason === "timeout";
	// The timer runs until the body has been read: a server that answers with
	// headers and then stalls is covered by the same budget.
	try {
		let response: Response;
		try {
			response = await fetch(path, {
				...rest,
				headers: { Accept: "application/json", ...headers },
				signal: controller.signal,
			});
		} catch (failure) {
			if (timedOut()) throw new ApiError("Request timed out", 0);
			if (failure instanceof DOMException && failure.name === "AbortError") throw failure;
			throw new ApiError("Server unreachable", 0);
		}
		if (!response.ok) {
			const failure = await errorFromResponse(response);
			if (failure.isUnauthorized && !unauthorizedIsExpected.has(path)) unauthorizedHandler?.();
			throw failure;
		}
		if (response.status === 204) return undefined as T;
		if (!isJSON(response)) throw new ApiError("Unexpected response from server", response.status);
		try {
			return (await response.json()) as T;
		} catch {
			if (timedOut()) throw new ApiError("Request timed out", 0);
			throw new ApiError("Unexpected response from server", response.status);
		}
	} finally {
		clearTimeout(timer);
		signal?.removeEventListener("abort", forwardAbort);
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

/** `GET /api/health`: liveness and the running version (FR-41, FR-41a). */
export interface Health {
	status: string;
	/** Release tag such as `v1.1.0`, or `dev` for a development build. */
	version: string;
	uptimeSec: number;
}

export function getHealth(): Promise<Health> {
	return request<Health>("/api/health");
}

export function getStatusHistory(cardID: string): Promise<StatusChange[]> {
	return request<StatusChange[]>(`/api/cards/${encodeURIComponent(cardID)}/status/history`);
}

export function connectStatusEvents(
	onStatusChange: (event: StatusEvent) => void,
	onRunRecorded: (event: RunEvent) => void = () => {},
): EventSource | undefined {
	if (typeof EventSource === "undefined") return undefined;
	const source = new EventSource("/api/events");
	// Malformed transient events are ignored; REST reads remain available.
	source.addEventListener("status.changed", (event) => {
		const data = eventData<StatusEvent>(event);
		if (data?.cardId && data.snapshot) onStatusChange(data);
	});
	source.addEventListener("run.recorded", (event) => {
		const data = eventData<RunEvent>(event);
		if (data?.cardId && data.run?.startedAt) onRunRecorded(data);
	});
	return source;
}

function eventData<T>(event: Event): T | undefined {
	if (!(event instanceof MessageEvent)) return undefined;
	try {
		return JSON.parse(event.data) as T;
	} catch {
		return undefined;
	}
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

/** A paired browser as listed on the Devices page (FR-55). */
export interface Device {
	id: string;
	name: string;
	pairedAt: string;
	lastSeenAt: string;
	/** True for the device that made the request. */
	current: boolean;
}

/**
 * Access state of this browser: `open` when the server runs without access
 * control (MARIONETTE_AUTH=off), `paired`, or `unpaired` — with `bootstrap`
 * set while no device is paired at all and the code is in the service log.
 */
export type Session =
	| { status: "open" }
	| { status: "paired"; device: Device; expiryDays: number }
	| { status: "unpaired"; bootstrap: boolean };

export async function getSession(): Promise<Session> {
	try {
		const session = await request<{ device: Device; expiryDays: number }>("/api/session");
		return { status: "paired", device: session.device, expiryDays: session.expiryDays };
	} catch (failure) {
		if (failure instanceof ApiError && failure.isUnauthorized) {
			return { status: "unpaired", bootstrap: failure.envelope?.bootstrap === true };
		}
		if (failure instanceof ApiError && failure.isNotFound) return { status: "open" };
		throw failure;
	}
}

export async function pairDevice(code: string, name: string): Promise<Device> {
	const paired = await request<{ device: Device }>("/api/pairing", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ code, name }),
	});
	return paired.device;
}

export interface PairingCode {
	/** Grouped for display, e.g. "K7QM-3XRD". */
	code: string;
	expiresAt: string;
}

export function createPairingCode(): Promise<PairingCode> {
	return request<PairingCode>("/api/pairing/code", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
	});
}

export function listDevices(): Promise<Device[]> {
	return request<Device[]>("/api/devices");
}

/** Renames a paired device (FR-57); a 422 carries the message in `fields.name`. */
export async function renameDevice(id: string, name: string): Promise<Device> {
	const renamed = await request<{ device: Device }>(`/api/devices/${encodeURIComponent(id)}`, {
		method: "PATCH",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ name }),
	});
	return renamed.device;
}

export function removeDevice(id: string): Promise<void> {
	return request<void>(`/api/devices/${encodeURIComponent(id)}`, {
		method: "DELETE",
		headers: { "Content-Type": "application/json" },
	});
}
