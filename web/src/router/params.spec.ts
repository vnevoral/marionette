import { describe, expect, it } from "vitest";
import { safeNext, singleParam } from "@/router/params";

describe("router params", () => {
	it("keeps only in-app paths as the target after pairing", () => {
		expect(safeNext("/cards/printer?tab=runs")).toBe("/cards/printer?tab=runs");
		expect(safeNext("//evil.example/x")).toBe("/");
		expect(safeNext("https://evil.example")).toBe("/");
		expect(safeNext(undefined)).toBe("/");
		expect(safeNext(["/devices"])).toBe("/");
	});

	it("collapses repeated params to the first value", () => {
		expect(singleParam(["a", "b"])).toBe("a");
		expect(singleParam(undefined)).toBe("");
	});
});
