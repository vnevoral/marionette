import js from "@eslint/js";
import vue from "eslint-plugin-vue";
import tseslint from "typescript-eslint";
import globals from "globals";

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
		rules: {
			"vue/html-indent": ["warn", "tab"],
		},
	},
	{
		ignores: ["dist/**", "node_modules/**"],
	},
);
