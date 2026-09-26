import { expect, test } from "@playwright/test";
import { createCard, deleteAllCards, expectNoHorizontalScroll } from "./helpers";

test.beforeEach(async ({ request }) => {
	await deleteAllCards(request);
});

test("rejects mutating requests from another site (NFR-12)", async ({ request, baseURL }) => {
	await createCard(request, { id: "target", name: "Target" });
	const run = "/api/cards/target/actions/primary";
	const json = { "Content-Type": "application/json" };

	const foreign = await request.post(run, { headers: { ...json, Origin: "https://evil.example" } });
	expect(foreign.status()).toBe(403);
	const crossSite = await request.post(run, {
		headers: { ...json, "Sec-Fetch-Site": "cross-site" },
	});
	expect(crossSite.status()).toBe(403);
	const form = await request.post("/api/cards", {
		headers: { "Content-Type": "text/plain" },
		data: "name=x",
	});
	expect(form.status()).toBe(415);
	const sameOrigin = await request.post(run, { headers: { ...json, Origin: baseURL! } });
	expect(sameOrigin.status()).toBe(202);
});

test("a page on another origin cannot trigger an action in the operator's browser", async ({
	page,
	baseURL,
}) => {
	await createCard(page.request, { id: "target", name: "Target" });
	// 127.0.0.1 and localhost are different origins for the browser.
	const foreignOrigin = baseURL!.replace("127.0.0.1", "localhost");
	await page.goto(`${foreignOrigin}/`);
	const status = await page.evaluate(async (target) => {
		try {
			const response = await fetch(`${target}/api/cards/target/actions/primary`, {
				method: "POST",
				mode: "no-cors",
			});
			return response.type;
		} catch {
			return "network-error";
		}
	}, baseURL!);
	expect(["opaque", "network-error"]).toContain(status);
	const runs = (await (await page.request.get("/api/cards/target/runs")).json()) as unknown[];
	expect(runs).toHaveLength(0);
});

test.describe("at 320 px", () => {
	test.use({ viewport: { width: 320, height: 640 } });

	test("overview, detail, edit and devices do not scroll horizontally", async ({
		page,
		request,
	}) => {
		await createCard(request, {
			id: "narrow",
			name: "A card with a fairly long name",
			withStatus: true,
		});
		for (const path of [
			"/",
			"/cards/narrow",
			"/cards/narrow/edit",
			"/cards/new/edit",
			"/devices",
		]) {
			await page.goto(path);
			await expect(page.locator("main.page")).toBeVisible();
			await expect(page.locator(".loading-state")).toHaveCount(0);
			await expectNoHorizontalScroll(page);
		}
	});

	test("the save notice fits the screen", async ({ page, request }) => {
		await createCard(request, { id: "narrow", name: "A card with a fairly long name" });
		await page.goto("/cards/narrow/edit");
		await page.getByLabel("Description", { exact: true }).fill("Saved on a phone");
		await page.getByRole("button", { name: "Save card" }).click();
		const notice = page.getByRole("alert").filter({ hasText: "Card saved" });
		await expect(notice).toBeVisible();
		// Palette colours from the preset, not Aura's bright green (AA contrast).
		await expect(notice.locator(".p-toast-summary")).toHaveCSS("color", "rgb(57, 114, 84)");
		const box = await notice.boundingBox();
		expect(box!.x).toBeGreaterThanOrEqual(0);
		expect(box!.x + box!.width).toBeLessThanOrEqual(320);
		await expectNoHorizontalScroll(page);
	});

	test("the pairing screen and a shown code fit the screen", async ({ page, browser, baseURL }) => {
		await page.goto("/devices");
		await page.getByRole("button", { name: "Pair a new device" }).click();
		await expect(page.getByLabel("Pairing code")).toBeVisible();
		await expectNoHorizontalScroll(page);

		const context = await browser.newContext({
			baseURL,
			viewport: { width: 320, height: 640 },
			storageState: { cookies: [], origins: [] },
		});
		const unpaired = await context.newPage();
		await unpaired.goto("/pair");
		await expect(unpaired.getByRole("button", { name: "Pair device" })).toBeVisible();
		await expectNoHorizontalScroll(unpaired);
		await context.close();
	});
});
