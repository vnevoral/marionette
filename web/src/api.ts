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
}

export interface Run {
	actionKind: string;
	startedAt: string;
	duration: number;
	exitCode: number;
	output: string;
	truncated: boolean;
	outcome: "ok" | "fail" | "timeout";
}

export interface StatusSnapshot {
	state: StatusState;
	checkedAt: string;
	lastCheck: Run;
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const response = await fetch(path, {
		headers: { Accept: "application/json", ...options?.headers },
		...options,
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
		body: JSON.stringify(card),
	});
}

export function updateCard(card: ActionCard): Promise<ActionCard> {
	return request<ActionCard>(`/api/cards/${encodeURIComponent(card.id)}`, {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(card),
	});
}

export function deleteCard(cardID: string): Promise<void> {
	return request(`/api/cards/${encodeURIComponent(cardID)}`, { method: "DELETE" });
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

export function enqueuePrimary(cardID: string): Promise<void> {
	return request(`/api/cards/${encodeURIComponent(cardID)}/actions/primary`, { method: "POST" });
}

export function enqueueStatus(cardID: string): Promise<void> {
	return request(`/api/cards/${encodeURIComponent(cardID)}/actions/status/check`, {
		method: "POST",
	});
}
