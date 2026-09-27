import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import DetailPanel from "@/components/DetailPanel.vue";

describe("DetailPanel", () => {
	it("draws a stripe in the card colour", () => {
		const panel = mount(DetailPanel, { props: { title: "Current status", color: "pink" } });
		expect(panel.classes()).toContain("has-color");
		expect(panel.attributes("data-color")).toBe("pink");
		expect(panel.attributes("style")).toContain("--card-stripe: var(--card-color-pink)");
	});

	it("has no stripe without a colour or with one outside the palette", () => {
		for (const color of [undefined, "", "red"]) {
			const panel = mount(DetailPanel, { props: { title: "Current status", color } });
			expect(panel.classes()).not.toContain("has-color");
			expect(panel.attributes("data-color")).toBeUndefined();
			expect(panel.attributes("style") ?? "").not.toContain("--card-stripe");
		}
	});
});
