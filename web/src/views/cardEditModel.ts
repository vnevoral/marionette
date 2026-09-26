// Pure helpers behind CardEditView: form defaults, payload normalisation and
// client-side validation. They have no Vue or router dependencies so they can
// be unit-tested directly (block 0030).
import type { Action, ActionCard } from "@/api";
import { DEFAULT_CARD_ICON } from "@/ui/icons";

/** Repeatable editor rows carry a stable id so Vue keys survive removal. */
export type ArgumentRow = { id: string; value: string };
export type EnvironmentRow = { id: string; key: string; value: string };

let rowSequence = 0;

export function newRowId(): string {
	rowSequence += 1;
	return `row-${rowSequence}`;
}

export function argumentRows(args: string[] | undefined): ArgumentRow[] {
	return (args ?? []).map((value) => ({ id: newRowId(), value }));
}

export function emptyAction(): Action {
	return { command: "", args: [], dir: "", env: {}, timeoutSec: 30, rule: { type: "exit_code" } };
}

export function emptyCard(): ActionCard {
	return {
		id: "",
		name: "",
		description: "",
		icon: DEFAULT_CARD_ICON,
		primary: emptyAction(),
	};
}

export function copyAction(action: Action): Action {
	return {
		...action,
		args: [...(action.args ?? [])],
		env: { ...(action.env ?? {}) },
		rule: { ...action.rule },
	};
}

export function environmentRows(env: Record<string, string> | undefined): EnvironmentRow[] {
	return Object.entries(env ?? {}).map(([key, value]) => ({ id: newRowId(), key, value }));
}

/** Builds the action payload sent to the API from the editor state: blank
 * arguments and environment rows are dropped, strings are trimmed and the
 * rule pattern is cleared for exit-code rules. */
export function actionFrom(
	action: Action,
	args: ArgumentRow[],
	environment: EnvironmentRow[],
): Action {
	const env = Object.fromEntries(
		environment.filter((row) => row.key.trim()).map((row) => [row.key.trim(), row.value]),
	);
	return {
		...action,
		args: args.map((row) => row.value.trim()).filter(Boolean),
		env,
		dir: action.dir?.trim(),
		rule: {
			...action.rule,
			pattern: action.rule.type === "exit_code" ? undefined : action.rule.pattern?.trim(),
		},
	};
}

export interface EditorState {
	form: ActionCard;
	statusEnabled: boolean;
	primaryArgs: ArgumentRow[];
	statusArgs: ArgumentRow[];
	primaryEnv: EnvironmentRow[];
	statusEnv: EnvironmentRow[];
}

/** Serialises the editable part of the form; two states with the same
 * fingerprint would produce the same save payload. */
export function fingerprint(state: EditorState): string {
	const { form } = state;
	return JSON.stringify({
		name: form.name,
		description: form.description,
		icon: form.icon,
		primary: actionFrom(form.primary, state.primaryArgs, state.primaryEnv),
		status:
			state.statusEnabled && form.status
				? actionFrom(form.status, state.statusArgs, state.statusEnv)
				: undefined,
		pollingIntervalSeconds: form.pollingIntervalSeconds,
		fastPollingIntervalSeconds: form.fastPollingIntervalSeconds,
		fastPollingWindowSeconds: form.fastPollingWindowSeconds,
	});
}

export interface ValidationResult {
	/** Field key → message; keys match the editor's `fieldErrors` map. */
	fieldErrors: Record<string, string>;
	/** The first message in field order, or "" when the form is valid. */
	firstError: string;
}

export function validate(form: ActionCard, statusEnabled: boolean): ValidationResult {
	const fieldErrors: Record<string, string> = {};
	let firstError = "";
	const addError = (key: string, message: string) => {
		fieldErrors[key] = message;
		if (!firstError) firstError = message;
	};
	if (!form.name.trim()) addError("name", "Card name is required");
	if (!form.primary.command.trim()) addError("primaryCommand", "Primary command is required");
	if (form.primary.timeoutSec <= 0) addError("primaryTimeout", "Primary timeout must be positive");
	if (statusEnabled) {
		if (!form.status?.command.trim()) addError("statusCommand", "Status command is required");
		if (!form.status || form.status.timeoutSec <= 0)
			addError("statusTimeout", "Status timeout must be positive");
	}
	return { fieldErrors, firstError };
}

/** Editor error keys of one action editor (primary or status). */
export interface ActionFieldErrors {
	command?: string;
	dir?: string;
	timeoutSec?: string;
	pattern?: string;
	args?: string;
	env?: string;
}

const actionFieldKeys: Record<string, keyof ActionFieldErrors> = {
	command: "command",
	dir: "dir",
	timeoutSec: "timeoutSec",
	"rule.pattern": "pattern",
	"rule.type": "pattern",
	args: "args",
	env: "env",
};

const editorFieldKeys: Record<keyof ActionFieldErrors, string> = {
	command: "Command",
	dir: "Dir",
	timeoutSec: "Timeout",
	pattern: "Pattern",
	args: "Args",
	env: "Env",
};

/**
 * Maps the `fields` of a 422 response (JSON paths such as `primary.command`
 * or `status.env.HOME`) onto the editor's error keys (`primaryCommand`,
 * `statusEnv`, `name`, …). Paths without an input of their own are returned
 * separately so the summary can still show them.
 */
export function fieldErrorsFromServer(fields: Record<string, string>): {
	fieldErrors: Record<string, string>;
	unmapped: string[];
} {
	const fieldErrors: Record<string, string> = {};
	const unmapped: string[] = [];
	for (const [path, message] of Object.entries(fields)) {
		const [scope, ...rest] = path.split(".");
		if (scope === "primary" || scope === "status") {
			const key = actionFieldKeys[rest.join(".")] ?? (rest[0] === "env" ? "env" : undefined);
			if (key) {
				const editorKey = `${scope}${editorFieldKeys[key]}`;
				if (!fieldErrors[editorKey]) fieldErrors[editorKey] = message;
				continue;
			}
		} else if (
			[
				"name",
				"description",
				"icon",
				"pollingIntervalSeconds",
				"fastPollingIntervalSeconds",
				"fastPollingWindowSeconds",
			].includes(path)
		) {
			fieldErrors[path] = message;
			continue;
		}
		unmapped.push(`${path}: ${message}`);
	}
	return { fieldErrors, unmapped };
}
