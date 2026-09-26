import { tmpdir } from "node:os";
import { join } from "node:path";
import { defineConfig, devices } from "@playwright/test";

// End-to-end tests against the real binary with the embedded SPA (blocks
// 0041, 0044). `make e2e` builds the binary first; the server starts with an
// empty configuration and access control on. The setup project pairs the
// browser with the code from the server log and saves the cookie, so every
// other spec runs as a paired device.
const port = Number(process.env.E2E_PORT ?? 18080);
process.env.E2E_SERVER_LOG ??= join(tmpdir(), `marionette-e2e-${port}.log`);
export const pairedState = "e2e/.auth/device.json";

export default defineConfig({
	testDir: "e2e",
	fullyParallel: false,
	workers: 1,
	forbidOnly: Boolean(process.env.CI),
	retries: 0,
	reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
	use: {
		baseURL: `http://127.0.0.1:${port}`,
		trace: "retain-on-failure",
	},
	projects: [
		{ name: "setup", testMatch: "**/*.setup.ts" },
		{
			name: "chromium",
			testMatch: "**/*.e2e.ts",
			dependencies: ["setup"],
			use: { ...devices["Desktop Chrome"], storageState: pairedState },
		},
	],
	webServer: {
		command: "bash e2e/start-server.sh",
		url: `http://127.0.0.1:${port}/api/health`,
		reuseExistingServer: false,
		timeout: 30_000,
		env: { E2E_PORT: String(port), E2E_SERVER_LOG: process.env.E2E_SERVER_LOG },
	},
});
