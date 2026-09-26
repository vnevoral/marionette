// Pure helpers behind CardEditView: form defaults, payload normalisation and
// client-side validation. They have no Vue or router dependencies so they can
// be unit-tested directly (block 0030).
import type { Action, ActionCard } from "@/api";
import { DEFAULT_CARD_ICON } from "@/ui/icons";
import { formatCommandLine, parseCommandLine } from "@/views/commandLine";

import type { EnvironmentRow } from "@/types";

export type { EnvironmentRow };

let rowSequence = 0;

export function newRowId(): string {
	rowSequence += 1;
	return `row-${rowSequence}`;
}

/** The command line the editor shows for a stored action (ADR-0012). */
export function commandLineOf(action: Action | undefined): string {
	return action ? formatCommandLine(action.command, action.args ?? []) : "";
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

/** Builds the action payload sent to the API from the editor state: the
 * command line is split into command and arguments (ADR-0012), blank
 * environment rows are dropped, strings are trimmed and the rule pattern is
 * cleared for exit-code rules. A line that does not parse is kept as the
 * command; validate() stops such a form from being saved. */
export function actionFrom(
	action: Action,
	commandLine: string,
	environment: EnvironmentRow[],
): Action {
	const parsed = parseCommandLine(commandLine);
	const env = Object.fromEntries(
		environment.filter((row) => row.key.trim()).map((row) => [row.key.trim(), row.value]),
	);
	return {
		...action,
		command: parsed.ok ? parsed.command : commandLine,
		args: parsed.ok ? parsed.args : [],
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
	primaryLine: string;
	statusLine: string;
	primaryEnv: EnvironmentRow[];
	statusEnv: EnvironmentRow[];
}

/** Serialises the editable part of the form; two states with the same
 * fingerprint would produce the same save payload (so an extra space in a
 * command line does not make the form dirty). */
export function fingerprint(state: EditorState): string {
	const { form } = state;
	return JSON.stringify({
		name: form.name,
		description: form.description,
		icon: form.icon,
		primary: actionFrom(form.primary, state.primaryLine, state.primaryEnv),
		status: state.statusEnabled
			? actionFrom(form.status ?? emptyAction(), state.statusLine, state.statusEnv)
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

/** Client-side check of one command line: required and parseable. */
function commandLineError(line: string, label: string): string {
	const parsed = parseCommandLine(line);
	if (!parsed.ok) return parsed.message;
	if (!parsed.command) return `${label} command is required`;
	return "";
}

export function validate(
	state: Pick<EditorState, "form" | "statusEnabled" | "primaryLine" | "statusLine">,
): ValidationResult {
	const { form, statusEnabled } = state;
	const fieldErrors: Record<string, string> = {};
	let firstError = "";
	const addError = (key: string, message: string) => {
		fieldErrors[key] = message;
		if (!firstError) firstError = message;
	};
	if (!form.name.trim()) addError("name", "Card name is required");
	const primaryError = commandLineError(state.primaryLine, "Primary");
	if (primaryError) addError("primaryCommand", primaryError);
	if (form.primary.timeoutSec <= 0) addError("primaryTimeout", "Primary timeout must be positive");
	if (statusEnabled) {
		const statusError = commandLineError(state.statusLine, "Status");
		if (statusError) addError("statusCommand", statusError);
		// The status editor shows emptyAction() until one of its fields changes.
		if ((form.status ?? emptyAction()).timeoutSec <= 0)
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
	env?: string;
}

const actionFieldKeys: Record<string, keyof ActionFieldErrors> = {
	command: "command",
	dir: "dir",
	timeoutSec: "timeoutSec",
	"rule.pattern": "pattern",
	"rule.type": "pattern",
	args: "command",
	env: "env",
};

const editorFieldKeys: Record<keyof ActionFieldErrors, string> = {
	command: "Command",
	dir: "Dir",
	timeoutSec: "Timeout",
	pattern: "Pattern",
	env: "Env",
};

/**
 * Maps the `fields` of a 422 response (JSON paths such as `primary.command`
 * or `status.env.HOME`) onto the editor's error keys (`primaryCommand`,
 * `statusEnv`, `name`, …). Paths without an input of their own are returned
 * separately so the summary can still show them.
 */
// `env.NAME` messages belong to the Environment block, which has no per-item
// error slot; `args[N]` messages belong to the command line (ADR-0012).
function actionFieldKey(rest: string[]): keyof ActionFieldErrors | undefined {
	const known = actionFieldKeys[rest.join(".")];
	if (known) return known;
	if (rest[0] === "env") return "env";
	if (rest[0]?.startsWith("args[")) return "command";
	return undefined;
}

export function fieldErrorsFromServer(fields: Record<string, string>): {
	fieldErrors: Record<string, string>;
	unmapped: string[];
} {
	const fieldErrors: Record<string, string> = {};
	const unmapped: string[] = [];
	for (const [path, message] of Object.entries(fields)) {
		const [scope, ...rest] = path.split(".");
		if (scope === "primary" || scope === "status") {
			const key = actionFieldKey(rest);
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

// The card created by the last save. After creating a card the edit view moves
// to its edit route; the router may reuse or recreate the component, and either
// way the form shows the saved card with "Card saved" without a reload.
let justSaved: ActionCard | undefined;

export function rememberSavedCard(card: ActionCard) {
	justSaved = card;
}

/** Returns and forgets the just-saved card when it matches `id`. */
export function takeSavedCard(id: string): ActionCard | undefined {
	if (justSaved?.id !== id) return undefined;
	const card = justSaved;
	justSaved = undefined;
	return card;
}
