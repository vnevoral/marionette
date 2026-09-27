import { describe, expect, it } from "vitest";
import { formatCommandLine, parseCommandLine, quoteWord } from "@/views/commandLine";

function parsed(line: string) {
	const result = parseCommandLine(line);
	if (!result.ok) throw new Error(`unexpected error: ${result.message}`);
	return [result.command, ...result.args];
}

function failure(line: string) {
	const result = parseCommandLine(line);
	if (result.ok) throw new Error(`expected an error for ${line}`);
	return result;
}

describe("parseCommandLine", () => {
	it("splits a simple command into the command and its arguments", () => {
		expect(parseCommandLine("/usr/bin/ping -c 1 -W 2 192.168.1.10")).toEqual({
			ok: true,
			command: "/usr/bin/ping",
			args: ["-c", "1", "-W", "2", "192.168.1.10"],
		});
	});

	it("ignores repeated, leading and trailing spaces and tabs", () => {
		expect(parsed("  echo \t a   b\t")).toEqual(["echo", "a", "b"]);
	});

	it("returns an empty command for an empty or blank line", () => {
		expect(parseCommandLine("")).toEqual({ ok: true, command: "", args: [] });
		expect(parseCommandLine("   \t ")).toEqual({ ok: true, command: "", args: [] });
	});

	it("keeps single-quoted text literally", () => {
		expect(parsed(`ls '/mnt/My Disk' 'a"b\\c $HOME | x'`)).toEqual([
			"ls",
			"/mnt/My Disk",
			'a"b\\c $HOME | x',
		]);
	});

	it('keeps double-quoted text literally except for \\" and \\\\', () => {
		expect(parsed(`echo "say \\"hi\\" \\\\ \\n $x | 'y'"`)).toEqual([
			"echo",
			`say "hi" \\ \\n $x | 'y'`,
		]);
	});

	it("takes the next character literally after a backslash outside quotes", () => {
		expect(parsed("ls My\\ Disk \\| \\$HOME \\\\")).toEqual(["ls", "My Disk", "|", "$HOME", "\\"]);
	});

	it("joins adjacent parts into one word", () => {
		expect(parsed(`echo a'b c'd "e f"g`)).toEqual(["echo", "ab cd", "e fg"]);
	});

	it("keeps empty quoted arguments", () => {
		expect(parsed(`printf '' ""`)).toEqual(["printf", "", ""]);
	});

	it("treats ~, * and ? as plain text", () => {
		expect(parsed("ls ~/data *.log file?")).toEqual(["ls", "~/data", "*.log", "file?"]);
	});

	it.each(["|", "&", ";", "<", ">", "(", ")", "`"])(
		"refuses an unquoted %s and explains how to use it",
		(char) => {
			const result = failure(`ping host ${char} x`);
			expect(result.index).toBe(10);
			expect(result.message).toContain(`"${char}" needs a shell`);
			expect(result.message).toContain("/bin/sh -c");
			expect(parsed(`echo '${char}'`)).toEqual(["echo", char]);
			expect(parsed(`echo "${char}"`)).toEqual(["echo", char]);
			expect(parsed(`echo \\${char}`)).toEqual(["echo", char]);
		},
	);

	it("refuses an unquoted $ with its own explanation", () => {
		const result = failure("echo $HOME");
		expect(result.index).toBe(5);
		expect(result.message).toContain("does not expand variables");
	});

	it("reports a missing closing quote at the opening quote", () => {
		expect(failure("echo 'abc")).toMatchObject({ index: 5, message: "Missing closing ' quote." });
		expect(failure('echo a"bc')).toMatchObject({ index: 6, message: 'Missing closing " quote.' });
		expect(failure('echo "abc\\"')).toMatchObject({ message: 'Missing closing " quote.' });
	});

	it("reports a backslash at the end", () => {
		expect(failure("echo abc\\")).toMatchObject({ index: 8 });
	});

	it("keeps a line break inside quotes", () => {
		expect(parsed("sh -c 'echo a\necho b'")).toEqual(["sh", "-c", "echo a\necho b"]);
		expect(parsed('sh -c "x\ny"')).toEqual(["sh", "-c", "x\ny"]);
	});

	it("refuses a line break outside quotes", () => {
		expect(failure("echo a\necho b").message).toBe("The command must be a single line.");
		expect(failure("echo a\r").index).toBe(6);
	});
});

describe("formatCommandLine", () => {
	it("writes plain words without quotes", () => {
		expect(formatCommandLine("/usr/bin/ping", ["-c", "1", "192.168.1.10"])).toBe(
			"/usr/bin/ping -c 1 192.168.1.10",
		);
		expect(
			formatCommandLine("/usr/bin/wakeonlan", ["-i", "192.168.1.255", "AA:BB:CC:DD:EE:FF"]),
		).toBe("/usr/bin/wakeonlan -i 192.168.1.255 AA:BB:CC:DD:EE:FF");
	});

	it("quotes words that need it", () => {
		expect(quoteWord("")).toBe("''");
		expect(quoteWord("My Disk")).toBe("'My Disk'");
		expect(quoteWord("it's")).toBe(`'it'\\''s'`);
		expect(quoteWord("a|b")).toBe("'a|b'");
	});

	it("returns an empty line for an empty action", () => {
		expect(formatCommandLine("", [])).toBe("");
		expect(formatCommandLine("true")).toBe("true");
	});

	it.each([
		["echo", ["hello e2e"]],
		["ls", ["/mnt/My Disk", "it's", 'say "hi"', "back\\slash", "$HOME", "a|b", "", "~", "*"]],
		["/bin/sh", ["-c", "ping -c 1 host | grep ttl && echo 'up'"]],
		["/bin/sh", ["-c", "echo a\necho b\r\n"]],
		["printf", ["%s\t%s", "tab\there", "(x)", "`y`", ";", "&", "<>"]],
		["", ["only args"]],
		["cmd with space", []],
	])("reads back %s %j unchanged", (command, args) => {
		const result = parseCommandLine(formatCommandLine(command, args));
		expect(result).toEqual({ ok: true, command, args });
	});
});
