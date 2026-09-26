export type StatusState = "unknown" | "ok" | "fail";

export const CARD_ICON_OPTIONS = [
	{ label: "Desktop", value: "pi pi-desktop" },
	{ label: "Home", value: "pi pi-home" },
	{ label: "Server", value: "pi pi-server" },
	{ label: "Cloud", value: "pi pi-cloud" },
	{ label: "Database", value: "pi pi-database" },
	{ label: "Globe", value: "pi pi-globe" },
	{ label: "Bolt", value: "pi pi-bolt" },
	{ label: "Cog", value: "pi pi-cog" },
	{ label: "Shield", value: "pi pi-shield" },
	{ label: "Heart", value: "pi pi-heart" },
] as const;

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
	duration: number;
	exitCode: number;
	output: string;
	truncated: boolean;
	outcome: "ok" | "fail" | "timeout" | "canceled";
}

export interface StatusSnapshot {
	state: StatusState;
	checkedAt: string;
	lastCheck: Run;
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

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	// Spread options first: a later spread would replace the merged headers and
	// drop Accept on every call that sets its own Content-Type.
	const response = await fetch(path, {
		...options,
		headers: { Accept: "application/json", ...options?.headers },
	});
	if (!response.ok) {
		let detail = `Request failed (${response.status})`;
		try {
			const error = (await response.json()) as { error?: string };
			detail = error.error ?? detail;
		} catch {
			// Keep the HTTP status when the server response is not JSON.
		}
		throw new Error(detail);
	}
	if (response.status === 204) {
		return undefined as T;
	}
	return (await response.json()) as T;
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
