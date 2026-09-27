import { expect, test, type Browser, type Page } from "@playwright/test";

// Pairing and device management with access control on (FR-50..FR-55,
// FR-57). The
// suite's own browser stays paired; extra browsers are fresh contexts.

/** A browser that was never paired: contexts inherit the project's saved
 * cookie unless the storage state is replaced explicitly. */
async function freshBrowser(browser: Browser, baseURL: string) {
	return browser.newContext({ baseURL, storageState: { cookies: [], origins: [] } });
}

/** Creates a code on the Devices page of `page` and returns it. */
async function codeFromDevicesPage(page: Page): Promise<string> {
	await page.goto("/devices");
	await page.getByRole("button", { name: "Pair a new device" }).click();
	const code = (await page.getByLabel("Pairing code").textContent())!.trim();
	expect(code).toMatch(/^[0-9A-Z]{4}-[0-9A-Z]{4}$/);
	return code;
}

async function pairWithCode(page: Page, code: string, name: string) {
	await page.goto("/pair");
	await page.getByLabel("Pairing code").fill(code);
	await page.getByLabel("Device name").fill(name);
	await page.getByRole("button", { name: "Pair device" }).click();
	await expect(page.getByRole("heading", { name: "Overview", level: 1 })).toBeVisible();
}

test("an unpaired browser gets only the pairing screen and no data", async ({
	browser,
	baseURL,
}) => {
	const context = await freshBrowser(browser, baseURL!);
	const page = await context.newPage();
	await page.goto("/devices");
	await expect(page).toHaveURL(/\/pair\?next=(%2F|\/)devices$/);
	// A device is paired already, so the log hint is replaced by the Devices hint.
	await expect(page.getByText("Devices → Pair a new device")).toBeVisible();
	await expect(page.getByText("journalctl")).toHaveCount(0);
	for (const path of ["/api/cards", "/api/devices", "/api/events"]) {
		expect((await context.request.get(path)).status(), path).toBe(401);
	}
	expect((await context.request.get("/api/health")).status()).toBe(200);
	await context.close();
});

test("a wrong code is refused with one message", async ({ browser, baseURL }) => {
	const context = await freshBrowser(browser, baseURL!);
	const page = await context.newPage();
	await page.goto("/pair");
	await page.getByLabel("Pairing code").fill("zzzz-zzzz");
	await expect(page.getByLabel("Pairing code")).toHaveValue("ZZZZ-ZZZZ");
	await page.getByRole("button", { name: "Pair device" }).click();
	await expect(page.getByRole("alert").filter({ hasText: "invalid or has expired" })).toBeVisible();
	await expect(page).toHaveURL(/\/pair/);
	await context.close();
});

test("pairs another browser with a code from Devices, then removes it", async ({
	page,
	browser,
	baseURL,
}) => {
	const code = await codeFromDevicesPage(page);
	const context = await freshBrowser(browser, baseURL!);
	const second = await context.newPage();
	await pairWithCode(second, code, "Second browser");
	await expect(second.getByRole("alert").filter({ hasText: "Device paired" })).toBeVisible();

	// The Devices page notices the new device without a reload.
	await expect(page.getByText('"Second browser" is now paired.')).toBeVisible({ timeout: 10_000 });
	const row = page.locator(".device-row").filter({ hasText: "Second browser" });
	await expect(row).toBeVisible();
	await expect(page.locator(".device-row").filter({ hasText: "E2E browser" })).toContainText(
		"This device",
	);

	await row.getByRole("button", { name: "Remove device Second browser" }).click();
	const dialog = page.getByRole("alertdialog");
	await expect(dialog).toContainText('Remove "Second browser"?');
	await dialog.getByRole("button", { name: "Remove device" }).click();
	await expect(page.getByText('"Second browser" was removed.')).toBeVisible();

	// The removed browser is sent back to pairing on its next request.
	await second.reload();
	await expect(second).toHaveURL(/\/pair/);
	await context.close();
});

test("a pairing link fills the code in", async ({ page, browser, baseURL }) => {
	await codeFromDevicesPage(page);
	const link = await page.getByLabel("Pairing link").inputValue();
	const context = await freshBrowser(browser, baseURL!);
	const other = await context.newPage();
	await other.goto(link);
	await expect(other.getByLabel("Pairing code")).toHaveValue(/^[0-9A-Z]{4}-[0-9A-Z]{4}$/);
	await other.getByRole("button", { name: "Pair device" }).click();
	await expect(other.getByRole("heading", { name: "Overview", level: 1 })).toBeVisible();

	// Removing the device you are using signs it out.
	await other.goto("/devices");
	const own = other.locator(".device-row").filter({ hasText: "This device" });
	await own.getByRole("button", { name: /^Remove device / }).click();
	const dialog = other.getByRole("alertdialog");
	await expect(dialog).toContainText("This is the device you are using");
	await dialog.getByRole("button", { name: "Remove device" }).click();
	await expect(other).toHaveURL(/\/pair/);
	await expect(
		other.getByRole("alert").filter({ hasText: "This device was removed" }),
	).toBeVisible();
	expect((await context.request.get("/api/cards")).status()).toBe(401);
	await context.close();
});

test("a browser renames itself and the new name survives a reload", async ({
	page,
	browser,
	baseURL,
}) => {
	// A separate browser, so the suite's own "E2E browser" keeps its name.
	const code = await codeFromDevicesPage(page);
	const context = await freshBrowser(browser, baseURL!);
	const other = await context.newPage();
	await pairWithCode(other, code, "Suggested name");

	await other.goto("/devices");
	const own = other.locator(".device-row").filter({ hasText: "This device" });
	await own.getByRole("button", { name: "Rename Suggested name" }).click();
	// While editing, the row shows the field in place of the name and tag.
	const field = other.getByLabel("Device name");
	await expect(field).toHaveValue("Suggested name");
	await field.fill("   ");
	await field.press("Enter");
	await expect(other.getByText("Device name is required")).toBeVisible();
	await field.fill("Renamed browser");
	await field.press("Enter");
	await expect(other.getByText('"Renamed browser" saved.')).toBeVisible();
	await expect(other.getByLabel("Device name")).toHaveCount(0);
	await expect(own).toContainText("Renamed browser");

	await other.reload();
	await expect(other.locator(".device-row").filter({ hasText: "This device" })).toContainText(
		"Renamed browser",
	);
	// Every paired device sees the new name.
	await page.goto("/devices");
	await expect(page.locator(".device-row").filter({ hasText: "Renamed browser" })).toBeVisible();
	await expect(page.locator(".device-row").filter({ hasText: "Suggested name" })).toHaveCount(0);

	// Clean up: the renamed browser removes itself.
	await other
		.locator(".device-row")
		.filter({ hasText: "This device" })
		.getByRole("button", { name: "Remove device Renamed browser" })
		.click();
	await other.getByRole("alertdialog").getByRole("button", { name: "Remove device" }).click();
	await expect(other).toHaveURL(/\/pair/);
	await context.close();
});
