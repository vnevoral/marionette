import { expect, type APIRequestContext, type Page } from "@playwright/test";

export interface CardInput {
	id: string;
	name: string;
	withStatus?: boolean;
}

/** Creates a card through the API as a non-browser client (no Origin header). */
export async function createCard(request: APIRequestContext, input: CardInput) {
	const action = { command: "true", timeoutSec: 5, rule: { type: "exit_code" } };
	const response = await request.post("/api/cards", {
		headers: { "Content-Type": "application/json" },
		data: {
			id: input.id,
			name: input.name,
			primary: {
				command: "echo",
				args: ["hello from", input.id],
				timeoutSec: 5,
				rule: { type: "exit_code" },
			},
			...(input.withStatus ? { status: action } : {}),
		},
	});
	expect(response.status()).toBe(201);
}

/** Removes every card so a spec starts from the empty state. */
export async function deleteAllCards(request: APIRequestContext) {
	const cards = (await (await request.get("/api/cards")).json()) as { id: string }[];
	for (const card of cards) {
		const response = await request.delete(`/api/cards/${encodeURIComponent(card.id)}`, {
			headers: { "Content-Type": "application/json" },
		});
		expect(response.status()).toBe(204);
	}
}

/** The action editor section whose heading is `title` ("Primary action", "Status action"). */
export function actionEditor(page: Page, title: string) {
	return page
		.locator("section.action-editor")
		.filter({ has: page.getByRole("heading", { name: title, exact: true }) });
}

/** The badge of the detail view's "Current status" panel. */
export function currentStatusBadge(page: Page) {
	return page
		.locator("section.detail-panel")
		.filter({ has: page.getByRole("heading", { name: "Current status" }) })
		.locator(".status-badge");
}

/** Asserts the page has no horizontal scroll (UX spec §9, 320 px). */
export async function expectNoHorizontalScroll(page: Page) {
	const overflow = await page.evaluate(
		() => document.documentElement.scrollWidth - document.documentElement.clientWidth,
	);
	expect(overflow, "page scrolls horizontally").toBeLessThanOrEqual(0);
}
