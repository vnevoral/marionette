import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import StatusBadge from "@/components/StatusBadge.vue";

function mountBadge(props: { label: string; icon: string; tone?: "healthy" | "problem" }) {
	return mount(StatusBadge, { props, global: { plugins: [PrimeVue] } });
}

describe("StatusBadge", () => {
	it("renders the label, the icon and the tone class", () => {
		const wrapper = mountBadge({ label: "Healthy", icon: "pi pi-check-circle", tone: "healthy" });
		expect(wrapper.text()).toBe("Healthy");
		expect(wrapper.find("i").classes()).toContain("pi-check-circle");
		expect(wrapper.find("i").attributes("aria-hidden")).toBe("true");
		expect(wrapper.classes()).toContain("status-badge");
		expect(wrapper.classes()).toContain("status-healthy");
	});

	it("defaults to the unknown tone", () => {
		const wrapper = mountBadge({ label: "Unknown", icon: "pi pi-question-circle" });
		expect(wrapper.classes()).toContain("status-unknown");
		expect(wrapper.classes()).not.toContain("status-problem");
	});
});
