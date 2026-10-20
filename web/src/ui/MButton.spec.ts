import { describe, expect, it } from "vitest";
import { mount, RouterLinkStub } from "@vue/test-utils";
import MButton from "@/ui/MButton.vue";

function mountButton(props: InstanceType<typeof MButton>["$props"]) {
	return mount(MButton, { props, global: { stubs: { RouterLink: RouterLinkStub } } });
}

describe("MButton", () => {
	it("is a plain button with its label and icon", async () => {
		const wrapper = mountButton({ label: "Run action", icon: "play", variant: "primary" });
		expect(wrapper.element.tagName).toBe("BUTTON");
		expect(wrapper.attributes("type")).toBe("button");
		expect(wrapper.find(".m-button-label").text()).toBe("Run action");
		expect(wrapper.find("svg").html()).toMatch(/lucide-play/);
		expect(wrapper.classes()).toEqual(expect.arrayContaining(["m-button", "m-button-primary"]));
		await wrapper.trigger("click");
		expect(wrapper.emitted("click")).toHaveLength(1);
	});

	it("shows a spinner instead of the icon and is disabled while loading", () => {
		const wrapper = mountButton({ label: "Save", icon: "check", loading: true });
		expect(wrapper.attributes("disabled")).toBeDefined();
		expect(wrapper.attributes("aria-busy")).toBe("true");
		expect(wrapper.classes()).toContain("m-button-loading");
		expect(wrapper.find(".m-button-spinner").exists()).toBe(true);
		expect(wrapper.html()).not.toMatch(/lucide-check/);
	});

	it("is a router link with `to`", () => {
		const wrapper = mountButton({ label: "New card", icon: "plus", to: "/cards/new/edit" });
		expect(wrapper.findComponent(RouterLinkStub).props("to")).toBe("/cards/new/edit");
		expect(wrapper.find("button").exists()).toBe(false);
		expect(wrapper.find(".m-button-label").text()).toBe("New card");
	});

	it("is an icon button without a label", () => {
		const wrapper = mountButton({ icon: "x", variant: "ghost" });
		expect(wrapper.classes()).toContain("m-button-icon-only");
		expect(wrapper.find(".m-button-label").exists()).toBe(false);
	});
});
