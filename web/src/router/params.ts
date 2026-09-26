import type { RouteParamValue } from "vue-router";

// Route params are typed as string | string[]; the app only ever routes with a
// single value, so a repeated param collapses to its first entry.
export function singleParam(value: RouteParamValue | RouteParamValue[] | undefined): string {
	if (Array.isArray(value)) return value[0] ?? "";
	return value ?? "";
}

/** Where to continue after pairing: a path inside the app, never another
 * origin ("//evil.example" or "https://…" fall back to the overview). */
export function safeNext(next: unknown): string {
	return typeof next === "string" && next.startsWith("/") && !next.startsWith("//") ? next : "/";
}
