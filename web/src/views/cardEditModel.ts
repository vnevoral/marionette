// Pure helpers behind CardEditView: form defaults, payload normalisation and
// client-side validation. They have no Vue or router dependencies so they can
// be unit-tested directly (block 0030).
import { CARD_ICON_OPTIONS, type Action, type ActionCard } from "@/api";

export type EnvironmentRow = { key: string; value: string };

export function emptyAction(): Action {
	return { command: "", args: [], dir: "", env: {}, timeoutSec: 30, rule: { type: "exit_code" } };
}

export function emptyCard(): ActionCard {
	return {
		id: "",
		name: "",
		description: "",
		icon: CARD_ICON_OPTIONS[0].value,
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
	return Object.entries(env ?? {}).map(([key, value]) => ({ key, value }));
}

/** Builds the action payload sent to the API from the editor state: blank
 * arguments and environment rows are dropped, strings are trimmed and the
 * rule pattern is cleared for exit-code rules. */
export function actionFrom(action: Action, args: string[], environment: EnvironmentRow[]): Action {
	const env = Object.fromEntries(
		environment.filter((row) => row.key.trim()).map((row) => [row.key.trim(), row.value]),
	);
	return {
		...action,
		args: args.map((arg) => arg.trim()).filter(Boolean),
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
	primaryArgs: string[];
	statusArgs: string[];
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
