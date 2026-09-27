import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import CardIdentityTile from "@/components/CardIdentityTile.vue";

describe("CardIdentityTile", () => {
	it("shows the card icon", () => {
		const tile = mount(CardIdentityTile, { props: { icon: "pi pi-server" } });
		expect(tile.find("i").classes()).toEqual(["pi", "pi-server"]);
	});

	it("falls back to the default icon", () => {
		const tile = mount(CardIdentityTile);
		expect(tile.find("i").classes()).toEqual(["pi", "pi-desktop"]);
	});
});
