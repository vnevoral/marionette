import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import MIcon from "@/ui/MIcon.vue";

function svg(name?: string) {
	return mount(MIcon, { props: { name } }).find("svg");
}

describe("MIcon", () => {
	it("renders a registered icon, hidden from assistive technology", () => {
		const icon = svg("server");
		expect(icon.html()).toMatch(/lucide-server/);
		expect(icon.attributes("aria-hidden")).toBe("true");
	});

	it("shows the default card icon for an empty name", () => {
		expect(svg("").html()).toMatch(/lucide-monitor/);
		expect(svg(undefined).html()).toMatch(/lucide-monitor/);
	});

	it("shows the ghost for a name the UI does not know (ADR-0014)", () => {
		expect(svg("pi pi-home").html()).toMatch(/lucide-ghost/);
		expect(svg("rocket").html()).toMatch(/lucide-ghost/);
		expect(svg("constructor").html()).toMatch(/lucide-ghost/);
	});

	it("spins the loader", () => {
		expect(svg("loader-circle").classes()).toContain("m-icon-spin");
		expect(svg("server").classes()).not.toContain("m-icon-spin");
	});
});
