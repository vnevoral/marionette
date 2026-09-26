import { describe, expect, it } from "vitest";
import type { ActionCard } from "@/api";
import {
	actionFrom,
	argumentRows,
	copyAction,
	emptyAction,
	emptyCard,
	environmentRows,
	fieldErrorsFromServer,
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
			argumentRows([" -c ", "", "1", "   "]),
			[
				{ id: "a", key: " TZ ", value: "UTC" },
				{ id: "b", key: "", value: "ignored" },
				{ id: "c", key: "  ", value: "ignored" },
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
		expect(fingerprint(state({ primaryArgs: argumentRows([" ", ""]) }))).toBe(base);
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
		expect(environmentRows({ A: "1", B: "2" })).toMatchObject([
			{ key: "A", value: "1" },
			{ key: "B", value: "2" },
		]);
		expect(environmentRows(undefined)).toEqual([]);
	});
});

describe("argumentRows and environmentRows", () => {
	it("give every row a distinct id", () => {
		const args = argumentRows(["a", "b"]);
		const env = environmentRows({ A: "1", B: "2" });
		const ids = [...args, ...env].map((row) => row.id);
		expect(new Set(ids).size).toBe(4);
		expect(args.map((row) => row.value)).toEqual(["a", "b"]);
		expect(env.map((row) => [row.key, row.value])).toEqual([
			["A", "1"],
			["B", "2"],
		]);
	});
});

describe("fieldErrorsFromServer", () => {
	it("maps 422 field paths onto editor keys and keeps the rest for the summary", () => {
		const { fieldErrors, unmapped } = fieldErrorsFromServer({
			name: "action card name is required",
			"primary.command": "action command is required",
			"primary.timeoutSec": "must be between 1 and 3600",
			"primary.rule.pattern": "output rule pattern is required",
			"status.env.HOME": "environment variable name is too long",
			"status.env.PATH": "second env message",
			"status.args": "must have at most 64 items",
			"status.dir": "too long",
			fastPollingIntervalSeconds: "fast polling interval must be less than polling interval",
			id: "must contain only letters, digits, '-' and '_'",
			"primary.mystery": "unknown",
		});
		expect(fieldErrors).toEqual({
			name: "action card name is required",
			primaryCommand: "action command is required",
			primaryTimeout: "must be between 1 and 3600",
			primaryPattern: "output rule pattern is required",
			statusEnv: "environment variable name is too long",
			statusArgs: "must have at most 64 items",
			statusDir: "too long",
			fastPollingIntervalSeconds: "fast polling interval must be less than polling interval",
		});
		expect(unmapped).toEqual([
			"id: must contain only letters, digits, '-' and '_'",
			"primary.mystery: unknown",
		]);
	});
});
