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
	await primary.getByLabel("Command").fill("echo");
	await primary.getByRole("button", { name: "Add argument" }).click();
	await primary.getByLabel("Argument 1", { exact: true }).fill("hello e2e");
	await page.getByLabel("Configure status action").check();
	await actionEditor(page, "Status action").getByLabel("Command").fill("true");
	await page.getByRole("button", { name: "Save card" }).click();
	await expect(page.getByText("Card saved")).toBeVisible();
	await expect(page.getByRole("heading", { name: "Edit card", level: 1 })).toBeVisible();

	await page.getByRole("link", { name: "Back to card detail" }).click();
	await expect(page.getByRole("heading", { name: "Printer", level: 1 })).toBeVisible();
	await expect(currentStatusBadge(page)).toHaveText("Unknown");

	// A manual check waits for the new result and shows it.
	await page.getByRole("button", { name: "Check status" }).click();
	await expect(page.getByText("Status updated")).toBeVisible();
	await expect(currentStatusBadge(page)).toHaveText("Healthy");

	// Without automatic checks the primary action reports acceptance at once;
	// its run then appears in the history with its output.
	await page.getByRole("button", { name: "Run action" }).click();
	await expect(page.getByText("Action accepted")).toBeVisible();
	await expect(page.getByRole("button", { name: "Run action" })).toBeEnabled();
	await page.getByText("View output").first().click();
	await expect(page.getByText("hello e2e")).toBeVisible();

	await page.getByRole("button", { name: "Delete card" }).click();
	const dialog = page.getByRole("alertdialog");
	await expect(dialog).toContainText('Delete "Printer"?');
	await dialog.getByRole("button", { name: "Delete card" }).click();
	await expect(page).toHaveURL("/");
	await expect(page.getByRole("heading", { name: "No action cards yet" })).toBeVisible();
});

test("runs actions from the dashboard", async ({ page, request }) => {
	await createCard(request, { id: "lamp", name: "Lamp", withStatus: true });
	await page.goto("/");
	const card = page.locator(".action-card").filter({ hasText: "Lamp" });
	await expect(card.getByText("Not checked yet")).toBeVisible();

	await card.getByRole("button", { name: "Check status" }).click();
	await expect(card.getByText("Updated")).toBeVisible();
	await expect(card.locator(".status-badge")).toHaveText("Healthy");
	await expect(card.getByRole("button", { name: "Check status" })).toBeEnabled();

	await card.getByRole("button", { name: "Run action" }).click();
	await expect(card.getByText("Accepted")).toBeVisible();
	await expect(page.getByText("1 action card · 1 healthy")).toBeVisible();
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
