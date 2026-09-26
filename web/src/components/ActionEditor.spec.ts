import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import type { Action } from "@/api";
import ActionEditor from "@/components/ActionEditor.vue";
import type { ArgumentRow, EnvironmentRow } from "@/views/cardEditModel";

type EditorProps = {
	args: ArgumentRow[];
	environment: EnvironmentRow[];
	errors: Record<string, string>;
};

// Mounts the editor with v-model wiring so emitted rows flow back into props,
// as they do under the real parent.
function mountEditor(overrides: Partial<EditorProps> = {}) {
	const action: Action = {
		command: "ping",
		timeoutSec: 5,
		rule: { type: "match", pattern: "ok" },
	};
	const wrapper = mount(ActionEditor, {
		props: {
			modelValue: action,
			args: overrides.args ?? [{ id: "a1", value: "-c" }],
			environment: overrides.environment ?? [{ id: "e1", key: "TZ", value: "UTC" }],
			title: "Primary action",
			errors: overrides.errors,
			"onUpdate:args": (value: ArgumentRow[]) => void wrapper.setProps({ args: value }),
			"onUpdate:environment": (value: EnvironmentRow[]) =>
				void wrapper.setProps({ environment: value }),
		},
		global: { plugins: [PrimeVue] },
	});
	return wrapper;
}

function lastEmitted<T>(wrapper: ReturnType<typeof mountEditor>, event: string): T {
	const calls = wrapper.emitted(event);
	if (!calls?.length) throw new Error(`${event} was not emitted`);
	return calls[calls.length - 1][0] as T;
}

describe("ActionEditor", () => {
	it("adds and removes argument rows itself and keeps row ids stable", async () => {
		const wrapper = mountEditor();
		await wrapper
			.findAll("button")
			.find((b) => b.text() === "Add argument")!
			.trigger("click");
		const added = lastEmitted<ArgumentRow[]>(wrapper, "update:args");
		expect(added).toHaveLength(2);
		expect(added[0]).toEqual({ id: "a1", value: "-c" });
		expect(added[1].id).not.toBe("a1");
		expect(added[1].value).toBe("");

		await wrapper.find('button[aria-label="Remove argument 1"]').trigger("click");
		const remaining = lastEmitted<ArgumentRow[]>(wrapper, "update:args");
		expect(remaining).toEqual([added[1]]);

		await wrapper.find('input[aria-label="Argument 1"]').setValue("-n");
		expect(lastEmitted<ArgumentRow[]>(wrapper, "update:args")).toEqual([
			{ id: added[1].id, value: "-n" },
		]);
		expect(wrapper.findAll('input[aria-label^="Argument"]')).toHaveLength(1);
	});

	it("edits environment rows by id", async () => {
		const wrapper = mountEditor({
			environment: [
				{ id: "e1", key: "TZ", value: "UTC" },
				{ id: "e2", key: "LANG", value: "C" },
			],
		});
		await wrapper.find('input[aria-label="Variable 2 value"]').setValue("C.UTF-8");
		expect(lastEmitted<EnvironmentRow[]>(wrapper, "update:environment")).toEqual([
			{ id: "e1", key: "TZ", value: "UTC" },
			{ id: "e2", key: "LANG", value: "C.UTF-8" },
		]);
		await wrapper.find('button[aria-label="Remove variable 1"]').trigger("click");
		expect(lastEmitted<EnvironmentRow[]>(wrapper, "update:environment")).toEqual([
			{ id: "e2", key: "LANG", value: "C.UTF-8" },
		]);
		await wrapper
			.findAll("button")
			.find((b) => b.text() === "Add variable")!
			.trigger("click");
		const rows = lastEmitted<EnvironmentRow[]>(wrapper, "update:environment");
		expect(rows).toHaveLength(2);
		expect(rows[1]).toMatchObject({ key: "", value: "" });
	});

	it("emits the action with the changed field only", async () => {
		const wrapper = mountEditor();
		const command = wrapper
			.findAll("label")
			.find((label) => label.text().startsWith("Command"))!
			.find("input");
		await command.setValue("curl");
		expect(lastEmitted<Action>(wrapper, "update:modelValue")).toEqual({
			command: "curl",
			timeoutSec: 5,
			rule: { type: "match", pattern: "ok" },
		});
	});

	it("labels output rules in plain language and shows field errors", () => {
		const wrapper = mountEditor({
			errors: { command: "Command is required", pattern: "Pattern is invalid", env: "Too many" },
		});
		expect(wrapper.text()).toContain("Output matches pattern");
		expect(wrapper.text()).toContain("Regex pattern");
		const errors = wrapper.findAll(".field-error").map((node) => node.text());
		expect(errors).toEqual(["Command is required", "Pattern is invalid", "Too many"]);
	});

	it("gives every control an accessible name", () => {
		const wrapper = mountEditor();
		for (const input of wrapper.findAll("input")) {
			const element = input.element;
			const wrappedByLabel = element.closest("label")?.textContent?.trim();
			const named =
				element.getAttribute("aria-label") ||
				element.getAttribute("aria-labelledby") ||
				wrappedByLabel;
			expect(named, `input ${element.outerHTML} has no accessible name`).toBeTruthy();
		}
		const combobox = wrapper.find('[role="combobox"]');
		const labelledBy = combobox.attributes("aria-labelledby");
		expect(labelledBy).toBeTruthy();
		expect(wrapper.find(`[id="${labelledBy}"]`).text()).toBe("Output rule");
		for (const button of wrapper.findAll("button")) {
			const name = button.text() || button.attributes("aria-label");
			expect(name, `button ${button.html()} has no accessible name`).toBeTruthy();
		}
	});
});
