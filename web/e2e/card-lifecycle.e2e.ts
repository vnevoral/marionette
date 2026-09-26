import { expect, test } from "@playwright/test";
import { actionEditor, createCard, currentStatusBadge, deleteAllCards } from "./helpers";

test.beforeEach(async ({ request }) => {
	await deleteAllCards(request);
});

test("creates, runs, checks and deletes a card through the UI", async ({ page }) => {
	await page.goto("/");
	await expect(page.getByRole("heading", { name: "No action cards yet" })).toBeVisible();
	await expect(page.getByText("Live")).toBeVisible();

	await page.getByRole("link", { name: "Create your first card" }).click();
	await expect(page.getByRole("heading", { name: "New card", level: 1 })).toBeVisible();
	await page.getByLabel("Name", { exact: true }).fill("Printer");
	const primary = actionEditor(page, "Primary action");
	// One command line; the preview shows how it is split (ADR-0012).
	await primary.getByLabel("Command line").fill("echo 'hello e2e'");
	await expect(primary.getByRole("list", { name: "Command and arguments" })).toHaveText([/echo/]);
	await expect(primary.locator(".preview-arg")).toHaveText(["hello e2e"]);
	await page.getByLabel("Configure status action").check();
	await actionEditor(page, "Status action").getByLabel("Command line").fill("true");
	await page.getByRole("button", { name: "Save card" }).click();
	await expect(page.getByRole("alert").filter({ hasText: "Card saved" })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Edit card", level: 1 })).toBeVisible();

	await page.getByRole("link", { name: "Back to card detail" }).click();
	await expect(page.getByRole("heading", { name: "Printer", level: 1 })).toBeVisible();
	await expect(currentStatusBadge(page)).toHaveText("Unknown");

	// A manual check waits for the new result; the badge shows it and the
	// outcome is only announced to screen readers (UX spec §4).
	await page.getByRole("button", { name: "Check status" }).click();
	await expect(currentStatusBadge(page)).toHaveText("Healthy");
	await expect(page.locator(".action-note [role='status']")).toHaveText("Status updated");

	// Without automatic checks the primary action reports acceptance at once;
	// its run then appears in the history with its output.
	await page.getByRole("button", { name: "Run action" }).click();
	await expect(page.locator(".action-note-text")).toHaveText("Action accepted");
	await expect(page.getByRole("button", { name: "Run action" })).toBeEnabled();
	await page.getByText("View output").first().click();
	await expect(page.getByText("hello e2e")).toBeVisible();

	await page.getByRole("button", { name: "Delete card" }).click();
	const dialog = page.getByRole("alertdialog");
	await expect(dialog).toContainText('Delete "Printer"?');
	await dialog.getByRole("button", { name: "Delete card" }).click();
	await expect(page).toHaveURL("/");
	await expect(page.getByRole("alert").filter({ hasText: "Card deleted" })).toContainText(
		'"Printer"',
	);
	await expect(page.getByRole("heading", { name: "No action cards yet" })).toBeVisible();
});

test("runs actions from the dashboard", async ({ page, request }) => {
	await createCard(request, { id: "lamp", name: "Lamp", withStatus: true });
	await page.goto("/");
	const card = page.locator(".action-card").filter({ hasText: "Lamp" });
	const note = card.locator(".action-note-text");
	await expect(note).toHaveText("Not checked yet");
	const height = (await card.boundingBox())?.height;

	// The badge carries the outcome; no second "Updated" line appears and the
	// card keeps its height (UX spec §4, block 0049).
	await card.getByRole("button", { name: "Check status" }).click();
	await expect(card.locator(".status-badge")).toHaveText("Healthy");
	await expect(card.getByRole("button", { name: "Check status" })).toBeEnabled();
	await expect(note).toHaveText(/^Last checked /);
	await expect(card.getByRole("status")).toHaveText("Updated");
	expect((await card.boundingBox())?.height).toBe(height);

	await card.getByRole("button", { name: "Run action" }).click();
	await expect(note).toHaveText("Accepted");
	expect((await card.boundingBox())?.height).toBe(height);
	await expect(page.getByText("1 action card · 1 healthy")).toBeVisible();
});

test("shows the output of a failed status check in the detail", async ({ page, request }) => {
	await createCard(request, {
		id: "broken",
		name: "Broken",
		statusCommand: { command: "ls", args: ["/marionette-e2e-missing"] },
	});
	await page.goto("/cards/broken");
	await page.getByRole("button", { name: "Check status" }).click();
	await expect(currentStatusBadge(page)).toHaveText("Problem");

	// FR-21a: the cause is visible without reading the API.
	const summary = page
		.locator("section.detail-panel")
		.filter({ has: page.getByRole("heading", { name: "Current status" }) });
	await expect(summary.locator("dt", { hasText: "Exit code" }).locator("+ dd")).not.toHaveText("0");
	await summary.getByText("View output").click();
	await expect(summary.locator("pre")).toContainText("/marionette-e2e-missing");
});

test("edits a stored action as one command line", async ({ page, request }) => {
	// createCard stores args ["hello from", id]: the space needs quotes.
	await createCard(request, { id: "quoted", name: "Quoted" });
	await page.goto("/cards/quoted/edit");
	const primary = actionEditor(page, "Primary action");
	const line = primary.getByLabel("Command line");
	await expect(line).toHaveValue("echo 'hello from' quoted");

	// A pipe needs a shell: refused beside the field, nothing is saved.
	await line.fill("echo hi | grep hi");
	await expect(primary.locator(".field-error")).toContainText('"|" needs a shell');
	await page.getByRole("button", { name: "Save card" }).click();
	await expect(page.getByRole("alert").filter({ hasText: "Card saved" })).toHaveCount(0);

	await line.fill("echo 'hello again' done");
	await page.getByRole("button", { name: "Save card" }).click();
	await expect(page.getByRole("alert").filter({ hasText: "Card saved" })).toBeVisible();
	const stored = (await (await request.get("/api/cards/quoted")).json()) as {
		primary: { command: string; args: string[] };
	};
	expect(stored.primary).toMatchObject({ command: "echo", args: ["hello again", "done"] });
});

test("keeps the form and shows the server's field error on an invalid save", async ({
	page,
	request,
}) => {
	await createCard(request, { id: "polled", name: "Polled", withStatus: true });
	await page.goto("/cards/polled/edit");
	await page.getByLabel("Polling interval (seconds)").fill("10");
	await page.getByLabel("Fast interval (seconds)").fill("20");
	await page.getByLabel("Description").fill("typed before the error");
	await page.getByRole("button", { name: "Save card" }).click();
	// The message appears beside the field and in the summary (UX spec §7.3).
	const message = "fast polling interval must be less than polling interval";
	await expect(page.locator(".field-error", { hasText: message })).toBeVisible();
	await expect(page.locator(".edit-feedback")).toContainText(message);
	await expect(page.getByLabel("Description")).toHaveValue("typed before the error");
	await expect(page.getByLabel("Fast interval (seconds)")).toHaveValue("20");
});

test("asks before discarding unsaved changes", async ({ page, request }) => {
	await createCard(request, { id: "guarded", name: "Guarded" });
	await page.goto("/cards/guarded/edit");
	await page.getByLabel("Name", { exact: true }).fill("Guarded (edited)");
	await page.getByRole("link", { name: "Back to card detail" }).click();

	const dialog = page.getByRole("alertdialog");
	await expect(dialog).toContainText("Discard unsaved changes?");
	await dialog.getByRole("button", { name: "Keep editing" }).click();
	await expect(page.getByLabel("Name", { exact: true })).toHaveValue("Guarded (edited)");

	await page.getByRole("link", { name: "Back to card detail" }).click();
	await page.getByRole("alertdialog").getByRole("button", { name: "Discard changes" }).click();
	await expect(page.getByRole("heading", { name: "Guarded", level: 1 })).toBeVisible();
});
