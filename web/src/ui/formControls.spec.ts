import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import { SelectRoot } from "reka-ui";
import MInput from "@/ui/MInput.vue";
import MNumberInput from "@/ui/MNumberInput.vue";
import MSelect from "@/ui/MSelect.vue";
import MSwitch from "@/ui/MSwitch.vue";

const colors = [
	{ label: "None", value: "" },
	{ label: "Teal", value: "teal" },
] as const;

describe("MInput", () => {
	it("passes attributes to the input and marks it invalid", async () => {
		const wrapper = mount(MInput, {
			props: { modelValue: "a", invalid: true },
			attrs: { id: "name", maxlength: "64" },
		});
		expect(wrapper.element.tagName).toBe("INPUT");
		expect(wrapper.attributes()).toMatchObject({
			id: "name",
			maxlength: "64",
			"aria-invalid": "true",
		});
		await wrapper.find("input").setValue("ab");
		expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual(["ab"]);
	});
});

describe("MNumberInput", () => {
	it("emits numbers and an empty field as undefined", async () => {
		const wrapper = mount(MNumberInput, { props: { modelValue: 5, min: 1 } });
		expect((wrapper.element as HTMLInputElement).value).toBe("5");
		await wrapper.find("input").setValue("12");
		expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual([12]);
		await wrapper.find("input").setValue("");
		expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual([undefined]);
	});
});

describe("MSelect", () => {
	it("shows the selected option and names the trigger", () => {
		const wrapper = mount(MSelect, {
			props: { options: colors, modelValue: "teal", ariaLabel: "Color" },
		});
		const trigger = wrapper.find("button");
		expect(trigger.attributes("role")).toBe("combobox");
		expect(trigger.attributes("aria-label")).toBe("Color");
		expect(trigger.text()).toContain("Teal");
	});

	it("keeps the empty option, which Reka UI cannot hold, as an empty string", async () => {
		const wrapper = mount(MSelect, { props: { options: colors, modelValue: "" } });
		expect(wrapper.find("button").text()).toContain("None");
		const root = wrapper.findComponent(SelectRoot);
		root.vm.$emit("update:modelValue", "teal");
		root.vm.$emit("update:modelValue", "\u0000empty");
		expect(wrapper.emitted("update:modelValue")).toEqual([["teal"], [""]]);
	});

	it("gives the value slot no option for a value outside the list", () => {
		const wrapper = mount(MSelect, {
			props: { options: colors, modelValue: "red" },
			slots: {
				value: `<template #value="{ option, value }">{{ option ? "known" : value }}</template>`,
			},
		});
		expect(wrapper.find("button").text()).toContain("red");
	});
});

describe("MSwitch", () => {
	it("is a switch that toggles its model", async () => {
		const wrapper = mount(MSwitch, { props: { id: "status-enabled", modelValue: false } });
		const root = wrapper.find("#status-enabled");
		expect(root.attributes("role")).toBe("switch");
		expect(root.attributes("aria-checked")).toBe("false");
		await root.trigger("click");
		expect(wrapper.emitted("update:modelValue")?.at(-1)).toEqual([true]);
	});
});
