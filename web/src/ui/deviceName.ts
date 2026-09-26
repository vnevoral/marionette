// Suggested name for a device being paired, e.g. "Chrome on Android"; the
// operator can change it before pairing.
const BROWSERS: [RegExp, string][] = [
	[/Edg\//, "Edge"],
	[/OPR\/|Opera/, "Opera"],
	[/Firefox\/|FxiOS/, "Firefox"],
	[/Chrome\/|CriOS/, "Chrome"],
	[/Safari\//, "Safari"],
];

const SYSTEMS: [RegExp, string][] = [
	[/iPhone/, "iPhone"],
	[/iPad/, "iPad"],
	[/Android/, "Android"],
	[/Windows/, "Windows"],
	[/Mac OS X|Macintosh/, "macOS"],
	[/CrOS/, "ChromeOS"],
	[/Linux/, "Linux"],
];

export function suggestDeviceName(userAgent: string): string {
	const browser = BROWSERS.find(([pattern]) => pattern.test(userAgent))?.[1];
	const system = SYSTEMS.find(([pattern]) => pattern.test(userAgent))?.[1];
	if (browser && system) return `${browser} on ${system}`;
	return browser ?? system ?? "This browser";
}
