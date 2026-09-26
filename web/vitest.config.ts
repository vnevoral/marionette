import { defineConfig, mergeConfig } from "vitest/config";
import viteConfig from "./vite.config";

// Unit tests share the Vite config (plugins, "@" alias) and run in happy-dom.
export default mergeConfig(
	viteConfig,
	defineConfig({
		test: {
			environment: "happy-dom",
			include: ["src/**/*.spec.ts"],
			coverage: {
				provider: "v8",
				include: ["src/**"],
				exclude: ["src/**/*.spec.ts", "src/env.d.ts", "src/main.ts"],
				reporter: ["text", "html"],
			},
		},
	}),
);
