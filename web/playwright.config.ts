import { defineConfig, devices } from "@playwright/test";

// End-to-end tests against the real binary with the embedded SPA (block 0041).
// `make e2e` builds the binary first; the server starts with an empty
// configuration, so the specs create what they need and run one at a time.
const port = Number(process.env.E2E_PORT ?? 18080);

export default defineConfig({
	testDir: "e2e",
	testMatch: "**/*.e2e.ts",
	fullyParallel: false,
	workers: 1,
	forbidOnly: Boolean(process.env.CI),
	retries: 0,
	reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
	use: {
		baseURL: `http://127.0.0.1:${port}`,
		trace: "retain-on-failure",
	},
	projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
	webServer: {
		command: "bash e2e/start-server.sh",
		url: `http://127.0.0.1:${port}/api/health`,
		reuseExistingServer: false,
		timeout: 30_000,
		env: { E2E_PORT: String(port) },
	},
});
