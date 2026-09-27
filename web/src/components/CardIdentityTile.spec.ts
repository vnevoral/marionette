import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import CardIdentityTile from "@/components/CardIdentityTile.vue";

describe("CardIdentityTile", () => {
	it("shows the icon with a stripe in the card colour", () => {
		const tile = mount(CardIdentityTile, { props: { icon: "pi pi-server", color: "pink" } });
		expect(tile.find("i").classes()).toEqual(["pi", "pi-server"]);
		expect(tile.classes()).toContain("has-color");
		expect(tile.attributes("data-color")).toBe("pink");
		expect(tile.attributes("style")).toContain("--card-stripe: var(--card-color-pink)");
	});

	it("has no stripe without a colour or with one outside the palette", () => {
		for (const color of [undefined, "", "red"]) {
			const tile = mount(CardIdentityTile, { props: { color } });
			expect(tile.find("i").classes()).toEqual(["pi", "pi-desktop"]);
			expect(tile.classes()).not.toContain("has-color");
			expect(tile.attributes("data-color")).toBeUndefined();
		}
	});
});
