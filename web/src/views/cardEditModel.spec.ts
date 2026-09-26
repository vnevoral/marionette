import { describe, expect, it } from "vitest";
import type { ActionCard } from "@/api";
import {
	actionFrom,
	copyAction,
	emptyAction,
	emptyCard,
	environmentRows,
	fingerprint,
	validate,
	type EditorState,
} from "@/views/cardEditModel";

function validForm(): ActionCard {
	return {
		...emptyCard(),
		name: "Card",
		primary: { ...emptyAction(), command: "true" },
	};
}

function state(overrides: Partial<EditorState> = {}): EditorState {
	return {
		form: validForm(),
		statusEnabled: false,
		primaryArgs: [],
		statusArgs: [],
		primaryEnv: [],
		statusEnv: [],
		...overrides,
	};
}

describe("actionFrom", () => {
	it("trims strings and drops blank arguments and environment rows", () => {
		const action = actionFrom(
			{
				...emptyAction(),
				command: "ping",
				dir: " /tmp ",
				rule: { type: "match", pattern: " ok " },
			},
			[" -c ", "", "1", "   "],
			[
				{ key: " TZ ", value: "UTC" },
				{ key: "", value: "ignored" },
				{ key: "  ", value: "ignored" },
			],
		);
		expect(action.args).toEqual(["-c", "1"]);
		expect(action.env).toEqual({ TZ: "UTC" });
		expect(action.dir).toBe("/tmp");
		expect(action.rule).toEqual({ type: "match", pattern: "ok" });
	});

	it("clears the pattern for exit-code rules", () => {
		const action = actionFrom(
			{ ...emptyAction(), command: "true", rule: { type: "exit_code", pattern: "stale" } },
			[],
			[],
		);
		expect(action.rule).toEqual({ type: "exit_code", pattern: undefined });
	});
});

describe("validate", () => {
	it("accepts a complete form", () => {
		expect(validate(validForm(), false)).toEqual({ fieldErrors: {}, firstError: "" });
	});

	it("reports every missing field and the first message", () => {
		const form = validForm();
		form.name = "  ";
		form.primary.command = "";
		form.primary.timeoutSec = 0;
		const result = validate(form, false);
		expect(result.firstError).toBe("Card name is required");
		expect(Object.keys(result.fieldErrors)).toEqual(["name", "primaryCommand", "primaryTimeout"]);
	});

	it("checks the status action only when it is enabled", () => {
		const form = validForm();
		expect(validate(form, false).firstError).toBe("");
		const enabled = validate(form, true);
		expect(enabled.fieldErrors).toMatchObject({
			statusCommand: "Status command is required",
			statusTimeout: "Status timeout must be positive",
		});
		form.status = { ...emptyAction(), command: "ping", timeoutSec: 5 };
		expect(validate(form, true).firstError).toBe("");
	});
});

describe("fingerprint", () => {
	it("is stable for equivalent states and changes with edits", () => {
		const base = fingerprint(state());
		expect(fingerprint(state())).toBe(base);
		expect(fingerprint(state({ primaryArgs: [" ", ""] }))).toBe(base);
		const edited = state();
		edited.form.name = "Renamed";
		expect(fingerprint(edited)).not.toBe(base);
	});

	it("ignores the status action while the toggle is off", () => {
		const withStatus = state();
		withStatus.form.status = { ...emptyAction(), command: "ping" };
		expect(fingerprint(withStatus)).toBe(fingerprint(state()));
		expect(fingerprint({ ...withStatus, statusEnabled: true })).not.toBe(fingerprint(state()));
	});
});

describe("copyAction and environmentRows", () => {
	it("returns independent copies", () => {
		const original = { ...emptyAction(), args: ["a"], env: { K: "v" } };
		const copy = copyAction(original);
		copy.args?.push("b");
		copy.env!.X = "y";
		copy.rule.type = "match";
		expect(original.args).toEqual(["a"]);
		expect(original.env).toEqual({ K: "v" });
		expect(original.rule.type).toBe("exit_code");
	});

	it("maps environment objects to editable rows", () => {
		expect(environmentRows({ A: "1", B: "2" })).toEqual([
			{ key: "A", value: "1" },
			{ key: "B", value: "2" },
		]);
		expect(environmentRows(undefined)).toEqual([]);
	});
});
