// One-line command editing (ADR-0012, FR-22a). The editor shows an action as
// a single command line and splits it into the command and its arguments
// before saving; the API, storage and execution keep the structured form.
// The grammar is a small subset of POSIX shell quoting with no expansion, so
// what the user sees is exactly what the process receives.

export interface ParsedCommand {
	command: string;
	args: string[];
}

export type CommandLineResult =
	({ ok: true } & ParsedCommand) | { ok: false; message: string; index: number };

/** Characters that mean something only to a shell; unquoted they are refused. */
const SHELL_CHARACTERS = new Set(["|", "&", ";", "<", ">", "(", ")", "`", "$"]);

/** Words made only of these characters are written without quotes. */
const PLAIN_WORD = /^[A-Za-z0-9_@%+=:,./-]+$/;

function shellCharacterMessage(char: string): string {
	if (char === "$") {
		return "Marionette does not expand variables. Quote it ('$') to pass it as text.";
	}
	return (
		`"${char}" needs a shell, and Marionette runs commands without one. ` +
		`Quote it ('${char}') to pass it as text, or use /bin/sh -c '…' for pipes and redirection.`
	);
}

/**
 * Splits a command line into the command and its arguments:
 * - spaces and tabs separate words;
 * - '…' keeps everything literally, "…" too except for \" and \\;
 * - outside quotes a backslash takes the next character literally;
 * - adjacent parts form one word, '' or "" is an empty argument;
 * - nothing is expanded, and unquoted shell characters are an error.
 * The first error is returned with the index of the offending character.
 */
export function parseCommandLine(line: string): CommandLineResult {
	const words: string[] = [];
	let word = "";
	let inWord = false;
	let index = 0;
	const fail = (message: string, at: number): CommandLineResult => ({
		ok: false,
		message,
		index: at,
	});

	while (index < line.length) {
		const char = line[index];
		if (char === "\n" || char === "\r") {
			return fail("The command must be a single line.", index);
		}
		if (char === " " || char === "\t") {
			if (inWord) words.push(word);
			word = "";
			inWord = false;
			index += 1;
			continue;
		}
		inWord = true;
		if (char === "'") {
			const end = line.indexOf("'", index + 1);
			if (end < 0) return fail("Missing closing ' quote.", index);
			word += line.slice(index + 1, end);
			index = end + 1;
			continue;
		}
		if (char === '"') {
			let cursor = index + 1;
			let closed = false;
			while (cursor < line.length) {
				const inner = line[cursor];
				if (inner === '"') {
					closed = true;
					break;
				}
				if (inner === "\\" && (line[cursor + 1] === '"' || line[cursor + 1] === "\\")) {
					word += line[cursor + 1];
					cursor += 2;
					continue;
				}
				word += inner;
				cursor += 1;
			}
			if (!closed) return fail('Missing closing " quote.', index);
			index = cursor + 1;
			continue;
		}
		if (char === "\\") {
			if (index + 1 >= line.length) {
				return fail("A backslash at the end has nothing to escape.", index);
			}
			word += line[index + 1];
			index += 2;
			continue;
		}
		if (SHELL_CHARACTERS.has(char)) return fail(shellCharacterMessage(char), index);
		word += char;
		index += 1;
	}
	if (inWord) words.push(word);

	const [command = "", ...args] = words;
	return { ok: true, command, args };
}

/** Quotes one word so that parseCommandLine reads it back unchanged. */
export function quoteWord(word: string): string {
	if (word === "") return "''";
	if (PLAIN_WORD.test(word)) return word;
	return `'${word.replaceAll("'", `'\\''`)}'`;
}

/** The command line shown for a stored action; the inverse of parseCommandLine. */
export function formatCommandLine(command: string, args: readonly string[] = []): string {
	if (command === "" && args.length === 0) return "";
	return [command, ...args].map(quoteWord).join(" ");
}
