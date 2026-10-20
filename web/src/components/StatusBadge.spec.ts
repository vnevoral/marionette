import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import StatusBadge from "@/components/StatusBadge.vue";

function mountBadge(props: { label: string; icon: string; tone?: "healthy" | "problem" }) {
	return mount(StatusBadge, { props });
}

describe("StatusBadge", () => {
	it("renders the label, the icon and the tone class", () => {
		const wrapper = mountBadge({ label: "Healthy", icon: "circle-check", tone: "healthy" });
		expect(wrapper.text()).toBe("Healthy");
		expect(wrapper.find("svg").html()).toMatch(/lucide-circle-check/);
		expect(wrapper.find("svg").attributes("aria-hidden")).toBe("true");
		expect(wrapper.classes()).toContain("status-badge");
		expect(wrapper.classes()).toContain("status-healthy");
	});

	it("defaults to the unknown tone", () => {
		const wrapper = mountBadge({ label: "Unknown", icon: "circle-question-mark" });
		expect(wrapper.classes()).toContain("status-unknown");
		expect(wrapper.classes()).not.toContain("status-problem");
	});
});
