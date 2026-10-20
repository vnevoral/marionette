import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import CardIdentityTile from "@/components/CardIdentityTile.vue";

describe("CardIdentityTile", () => {
	it("shows the card icon", () => {
		const tile = mount(CardIdentityTile, { props: { icon: "server" } });
		expect(tile.find("svg").html()).toMatch(/lucide-server/);
	});

	it("falls back to the default icon", () => {
		const tile = mount(CardIdentityTile);
		expect(tile.find("svg").html()).toMatch(/lucide-monitor/);
	});

	it("shows the ghost for an icon the UI no longer knows", () => {
		const tile = mount(CardIdentityTile, { props: { icon: "pi pi-server" } });
		expect(tile.find("svg").html()).toMatch(/lucide-ghost/);
	});
});
