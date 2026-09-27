import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import PrimeVue from "primevue/config";
import type { Action } from "@/api";
import ActionEditor from "@/components/ActionEditor.vue";
import type { EnvironmentRow } from "@/views/cardEditModel";

type EditorProps = {
	commandLine: string;
	environment: EnvironmentRow[];
	errors: Record<string, string>;
};

// Mounts the editor with v-model wiring so emitted values flow back into props,
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
			commandLine: overrides.commandLine ?? "ping -c 1",
			environment: overrides.environment ?? [{ id: "e1", key: "TZ", value: "UTC" }],
			title: "Primary action",
			errors: overrides.errors,
			"onUpdate:commandLine": (value: string) => void wrapper.setProps({ commandLine: value }),
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
	it("edits the command as one line and shows how it is split (ADR-0012)", async () => {
		const wrapper = mountEditor();
		const input = wrapper.find("input.command-line");
		expect((input.element as HTMLInputElement).value).toBe("ping -c 1");
		expect(wrapper.find(".preview-command").text()).toBe("ping");
		expect(wrapper.findAll(".preview-arg").map((item) => item.text())).toEqual(["-c", "1"]);

		await input.setValue("/usr/bin/ping -c 1 'My Disk' ''");
		expect(lastEmitted<string>(wrapper, "update:commandLine")).toBe(
			"/usr/bin/ping -c 1 'My Disk' ''",
		);
		expect(wrapper.find(".preview-command").text()).toBe("/usr/bin/ping");
		expect(wrapper.findAll(".preview-arg").map((item) => item.text())).toEqual([
			"-c",
			"1",
			"My Disk",
			"empty",
		]);
		expect(wrapper.emitted("update:modelValue")).toBeUndefined();
	});

	it("explains a line that does not parse instead of showing a preview", async () => {
		const wrapper = mountEditor({ commandLine: "ping host | grep ttl" });
		expect(wrapper.find(".command-preview li").exists()).toBe(false);
		expect(wrapper.find(".field-error").text()).toContain('"|" needs a shell');
		expect(wrapper.find("input.command-line").attributes("aria-invalid")).toBe("true");

		await wrapper.find("input.command-line").setValue("ping host");
		expect(wrapper.find(".field-error").exists()).toBe(false);
		expect(wrapper.find(".preview-command").text()).toBe("ping");
	});

	it("keeps a line break of a stored argument in a text area", async () => {
		const wrapper = mountEditor({ commandLine: "sh -c 'echo a\necho b'" });
		expect(wrapper.find("input.command-line").exists()).toBe(false);
		const area = wrapper.find<HTMLTextAreaElement>("textarea.command-line");
		expect(area.element.value).toBe("sh -c 'echo a\necho b'");
		expect(wrapper.findAll(".preview-arg").map((item) => item.text())).toEqual([
			"-c",
			"echo a\necho b",
		]);
		expect(wrapper.find(".field-error").exists()).toBe(false);

		// Removing the break keeps the text area, so the field keeps focus.
		await area.setValue("sh -c 'echo a'");
		expect(lastEmitted<string>(wrapper, "update:commandLine")).toBe("sh -c 'echo a'");
		expect(wrapper.find("textarea.command-line").exists()).toBe(true);
	});

	it("shows no preview for an empty line", () => {
		const wrapper = mountEditor({ commandLine: "  " });
		expect(wrapper.find(".command-preview li").exists()).toBe(false);
		expect(wrapper.find(".field-error").exists()).toBe(false);
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
		const directory = wrapper
			.findAll("label")
			.find((label) => label.text().startsWith("Working directory"))!
			.find("input");
		await directory.setValue("/tmp");
		expect(lastEmitted<Action>(wrapper, "update:modelValue")).toEqual({
			command: "ping",
			dir: "/tmp",
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
			const labelFor = element.id ? wrapper.find(`label[for="${element.id}"]`) : undefined;
			const labelText = labelFor?.exists() ? labelFor.text() : undefined;
			const named =
				element.getAttribute("aria-label") ||
				element.getAttribute("aria-labelledby") ||
				wrappedByLabel ||
				labelText;
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
