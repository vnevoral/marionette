import { writeFileSync } from "node:fs";
import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// The build output is embedded directly into the Go binary from internal/webui/dist.
const outDir = fileURLToPath(new URL("../internal/webui/dist", import.meta.url));

export default defineConfig({
	plugins: [
		vue(),
		// emptyOutDir removes the tracked .gitkeep that lets go:embed compile
		// without a UI build; put it back.
		{
			name: "keep-dist-dir",
			closeBundle() {
				writeFileSync(`${outDir}/.gitkeep`, "");
			},
		},
	],
	resolve: {
		alias: {
			"@": fileURLToPath(new URL("./src", import.meta.url)),
		},
	},
	build: {
		outDir,
		emptyOutDir: true,
	},
	server: {
		port: 5173,
		proxy: {
			"/api": {
				target: "http://localhost:8080",
				changeOrigin: true,
			},
		},
	},
});
