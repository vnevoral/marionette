import { readFileSync } from "node:fs";
import { expect, test as setup } from "@playwright/test";
import { pairedState } from "../playwright.config";

// Pairs the test browser like an operator pairs the first device: the code
// comes from the service log (FR-53).
function codeFromLog(): string | undefined {
	const log = readFileSync(process.env.E2E_SERVER_LOG!, "utf8");
	return [...log.matchAll(/code=([0-9A-Z]{4}-[0-9A-Z]{4})/g)].at(-1)?.[1];
}

setup("pair the test browser with the code from the service log", async ({ page }) => {
	await page.goto("/");
	await expect(page).toHaveURL(/\/pair\?next=(%2F|\/)$/);
	await expect(page.getByRole("heading", { name: "Pair this device", level: 1 })).toBeVisible();
	await expect(page.getByText('journalctl -u marionette | grep "pairing code"')).toBeVisible();
	await expect(page.getByRole("navigation", { name: "Primary navigation" })).toHaveCount(0);

	let code: string | undefined;
	await expect.poll(() => (code = codeFromLog()), { timeout: 10_000 }).toBeTruthy();
	await page.getByLabel("Pairing code").fill(code!);
	await page.getByLabel("Device name").fill("E2E browser");
	await page.getByRole("button", { name: "Pair device" }).click();

	await expect(page.getByRole("alert").filter({ hasText: "Device paired" })).toBeVisible();
	await expect(page.getByRole("heading", { name: "Overview", level: 1 })).toBeVisible();
	await expect(page.getByRole("link", { name: "Devices" })).toBeVisible();
	await page.context().storageState({ path: pairedState });
});
