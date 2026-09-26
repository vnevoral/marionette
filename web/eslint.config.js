import js from "@eslint/js";
import vue from "eslint-plugin-vue";
import tseslint from "typescript-eslint";
import globals from "globals";
import prettier from "eslint-config-prettier";

export default tseslint.config(
	js.configs.recommended,
	...tseslint.configs.recommended,
	...vue.configs["flat/recommended"],
	{
		languageOptions: {
			globals: globals.browser,
		},
	},
	{
		files: ["**/*.vue"],
		languageOptions: {
			parserOptions: {
				parser: tseslint.parser,
			},
		},
	},
	{
		files: ["src/**/*.spec.ts"],
		languageOptions: {
			globals: { ...globals.browser, ...globals.node },
		},
	},
	// Must stay last: disables every formatting rule that conflicts with Prettier.
	prettier,
	{
		ignores: ["dist/**", "node_modules/**", "coverage/**", "vite.config.js", "vite.config.d.ts"],
	},
);
