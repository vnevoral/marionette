import { expect, test } from "@playwright/test";
import { createCard, deleteAllCards, expectNoHorizontalScroll } from "./helpers";

test.beforeEach(async ({ request }) => {
	await deleteAllCards(request);
});

test("the footer shows the version the server reports (FR-41a)", async ({ page, request }) => {
	const { version } = (await (await request.get("/api/health")).json()) as { version: string };
	await page.goto("/");
	await expect(page.locator(".app-footer")).toHaveText(`Marionette ${version}`);
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

	test("header buttons share the full width and keep their labels on one line", async ({
		page,
		request,
	}) => {
		await createCard(request, { id: "narrow", name: "A card with a fairly long name" });
		for (const width of [320, 412]) {
			await page.setViewportSize({ width, height: 800 });
			for (const path of ["/", "/cards/narrow"]) {
				await page.goto(path);
				await expect(page.locator(".loading-state")).toHaveCount(0);
				const row = await page.locator(".page-header-row").boundingBox();
				const actions = page.locator(".page-header-actions");
				const box = await actions.boundingBox();
				expect(box!.width, `${path} at ${width} px`).toBeCloseTo(row!.width, 0);
				const buttons = await actions.locator(":scope > *").all();
				expect(buttons.length).toBe(2);
				const widths = [];
				for (const button of buttons) {
					widths.push((await button.boundingBox())!.width);
					const lines = await button.locator(".p-button-label").evaluate((label) => {
						const range = document.createRange();
						range.selectNodeContents(label);
						return new Set(Array.from(range.getClientRects()).map((rect) => Math.round(rect.top)))
							.size;
					});
					expect(lines, `${path} at ${width} px`).toBe(1);
				}
				// Side by side they are equally wide; stacked they are full width.
				expect(widths[0]).toBeCloseTo(widths[1], 0);
			}
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
