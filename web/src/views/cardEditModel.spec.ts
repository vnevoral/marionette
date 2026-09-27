import { describe, expect, it } from "vitest";
import type { ActionCard } from "@/api";
import {
	actionFrom,
	commandLineOf,
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

function checked(overrides: Partial<EditorState> = {}) {
	return validate({ ...state(), ...overrides });
}

function state(overrides: Partial<EditorState> = {}): EditorState {
	return {
		form: validForm(),
		statusEnabled: false,
		primaryLine: "true",
		statusLine: "",
		primaryEnv: [],
		statusEnv: [],
		...overrides,
	};
}

describe("actionFrom", () => {
	it("splits the command line, trims strings and drops blank environment rows", () => {
		const action = actionFrom(
			{
				...emptyAction(),
				command: "stale",
				dir: " /tmp ",
				rule: { type: "match", pattern: " ok " },
			},
			"  ping  -c 1 '' 'My Disk' ",
			[
				{ id: "a", key: " TZ ", value: "UTC" },
				{ id: "b", key: "", value: "ignored" },
				{ id: "c", key: "  ", value: "ignored" },
			],
		);
		expect(action.command).toBe("ping");
		expect(action.args).toEqual(["-c", "1", "", "My Disk"]);
		expect(action.env).toEqual({ TZ: "UTC" });
		expect(action.dir).toBe("/tmp");
		expect(action.rule).toEqual({ type: "match", pattern: "ok" });
	});

	it("clears the pattern for exit-code rules", () => {
		const action = actionFrom(
			{ ...emptyAction(), command: "true", rule: { type: "exit_code", pattern: "stale" } },
			"true",
			[],
		);
		expect(action.rule).toEqual({ type: "exit_code", pattern: undefined });
	});

	it("keeps a line that does not parse as the command so the form stays dirty", () => {
		const action = actionFrom(emptyAction(), "ping host | grep ttl", []);
		expect(action.command).toBe("ping host | grep ttl");
		expect(action.args).toEqual([]);
	});
});

describe("commandLineOf", () => {
	it("shows a stored action as one line and reads back the same action", () => {
		const stored = { ...emptyAction(), command: "/bin/ls", args: ["-l", "/mnt/My Disk"] };
		expect(commandLineOf(stored)).toBe("/bin/ls -l '/mnt/My Disk'");
		const saved = actionFrom(stored, commandLineOf(stored), []);
		expect([saved.command, saved.args]).toEqual([stored.command, stored.args]);
		expect(commandLineOf(undefined)).toBe("");

		// A multi-line script written through the API survives an unchanged save.
		const script = { ...emptyAction(), command: "sh", args: ["-c", "echo a\necho b"] };
		const resaved = actionFrom(script, commandLineOf(script), []);
		expect([resaved.command, resaved.args]).toEqual(["sh", ["-c", "echo a\necho b"]]);
		expect(checked({ primaryLine: commandLineOf(script) }).firstError).toBe("");
	});
});

describe("validate", () => {
	it("accepts a complete form", () => {
		expect(checked()).toEqual({ fieldErrors: {}, firstError: "" });
	});

	it("reports every missing field and the first message", () => {
		const form = validForm();
		form.name = "  ";
		form.primary.timeoutSec = 0;
		const result = checked({ form, primaryLine: "   " });
		expect(result.firstError).toBe("Card name is required");
		expect(Object.keys(result.fieldErrors)).toEqual(["name", "primaryCommand", "primaryTimeout"]);
	});

	it("reports a command line that does not parse beside the command", () => {
		const result = checked({ primaryLine: "ping host | grep ttl" });
		expect(result.fieldErrors.primaryCommand).toContain('"|" needs a shell');
		expect(checked({ primaryLine: "echo 'open" }).fieldErrors.primaryCommand).toBe(
			"Missing closing ' quote.",
		);
		expect(checked({ primaryLine: "'' -x" }).fieldErrors.primaryCommand).toBe(
			"Primary command is required",
		);
	});

	it("checks the status action only when it is enabled", () => {
		expect(checked({ statusLine: "ping |" }).firstError).toBe("");
		expect(checked({ statusEnabled: true }).fieldErrors).toEqual({
			statusCommand: "Status command is required",
		});
		const form = validForm();
		form.status = { ...emptyAction(), timeoutSec: 0 };
		expect(
			checked({ form, statusEnabled: true, statusLine: "ping -c 1 host" }).fieldErrors,
		).toEqual({ statusTimeout: "Status timeout must be positive" });
		expect(checked({ statusEnabled: true, statusLine: "ping -c 1 host" }).firstError).toBe("");
	});
});

describe("fingerprint", () => {
	it("is stable for equivalent states and changes with edits", () => {
		const base = fingerprint(state());
		expect(fingerprint(state())).toBe(base);
		expect(fingerprint(state({ primaryLine: "  true   " }))).toBe(base);
		expect(fingerprint(state({ primaryLine: "true -v" }))).not.toBe(base);
		const edited = state();
		edited.form.name = "Renamed";
		expect(fingerprint(edited)).not.toBe(base);
	});

	it("changes with the card colour and treats none and a missing colour alike", () => {
		const none = state();
		none.form.color = "";
		const missing = state();
		delete missing.form.color;
		expect(fingerprint(none)).toBe(fingerprint(missing));
		const colored = state();
		colored.form.color = "blue";
		expect(fingerprint(colored)).not.toBe(fingerprint(none));
	});

	it("ignores the status action while the toggle is off", () => {
		const withStatus = state();
		withStatus.form.status = { ...emptyAction(), command: "ping" };
		withStatus.statusLine = "ping";
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

describe("environmentRows", () => {
	it("give every row a distinct id", () => {
		const env = environmentRows({ A: "1", B: "2" });
		expect(new Set(env.map((row) => row.id)).size).toBe(2);
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
			"primary.args[2]": "must be at most 1024 characters",
			"primary.args[5]": "second argument message",
			"status.command": "action command is required",
			"status.dir": "too long",
			fastPollingIntervalSeconds: "fast polling interval must be less than polling interval",
			color: "must be empty or one of: black, blue",
			id: "must contain only letters, digits, '-' and '_'",
			"primary.mystery": "unknown",
		});
		expect(fieldErrors).toEqual({
			name: "action card name is required",
			primaryCommand: "action command is required",
			primaryTimeout: "must be between 1 and 3600",
			primaryPattern: "output rule pattern is required",
			statusEnv: "environment variable name is too long",
			// Arguments are part of the command line (ADR-0012); the first
			// message for the line wins.
			statusCommand: "must have at most 64 items",
			statusDir: "too long",
			fastPollingIntervalSeconds: "fast polling interval must be less than polling interval",
			color: "must be empty or one of: black, blue",
		});
		expect(unmapped).toEqual([
			"id: must contain only letters, digits, '-' and '_'",
			"primary.mystery: unknown",
		]);
	});
});
