import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { h } from "vue";
import { createMemoryHistory, createRouter } from "vue-router";
import PrimeVue from "primevue/config";
import {
	ApiError,
	createPairingCode,
	listDevices,
	removeDevice,
	renameDevice,
	type Device,
} from "@/api";
import { setSession, useSession } from "@/composables/useSession";
import { fakeConfirm } from "@/test/fakeConfirm";
import { fakeToast } from "@/test/fakeToast";
import DevicesView from "@/views/DevicesView.vue";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return {
		...actual,
		listDevices: vi.fn(),
		createPairingCode: vi.fn(),
		removeDevice: vi.fn(),
		renameDevice: vi.fn(),
	};
});

const laptop: Device = {
	id: "d1",
	name: "Laptop",
	pairedAt: "2026-09-20T10:00:00Z",
	lastSeenAt: "2026-09-26T10:00:00Z",
	current: true,
};
const phone: Device = { ...laptop, id: "d2", name: "Phone", current: false };
const tablet: Device = { ...laptop, id: "d3", name: "Tablet", current: false };

async function mountDevices() {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{ path: "/devices", component: DevicesView },
			{ path: "/pair", name: "pair", component: { render: () => h("div", "pair") } },
		],
	});
	await router.push("/devices");
	await router.isReady();
	const confirm = fakeConfirm();
	const toast = fakeToast();
	const wrapper = mount(DevicesView, {
		global: { plugins: [router, PrimeVue], provide: { ...confirm.provide, ...toast.provide } },
	});
	await flushPromises();
	return { wrapper, router, confirm, toast };
}

describe("DevicesView", () => {
	beforeEach(() => {
		setSession({ status: "paired", device: laptop, expiryDays: 30 });
		vi.mocked(listDevices).mockReset().mockResolvedValue([laptop, phone]);
		vi.mocked(createPairingCode).mockReset();
		vi.mocked(removeDevice).mockReset().mockResolvedValue(undefined);
		vi.mocked(renameDevice).mockReset();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it("lists devices with this device marked and the configured expiry", async () => {
		const { wrapper } = await mountDevices();
		const rows = wrapper.findAll(".device-row");
		expect(rows.map((row) => row.find(".device-name span").text())).toEqual(["Laptop", "Phone"]);
		expect(rows[0].find(".p-tag").text()).toBe("This device");
		expect(rows[1].find(".p-tag").exists()).toBe(false);
		expect(wrapper.text()).toContain("not used for 30 days must pair again");
		expect(rows[1].find(".device-remove-button").attributes("aria-label")).toBe(
			"Remove device Phone",
		);
		expect(rows[1].find(".device-rename-button").attributes("aria-label")).toBe("Rename Phone");
		wrapper.unmount();
	});

	it("removes another device after confirmation and says so inline", async () => {
		const { wrapper, confirm, toast } = await mountDevices();
		await wrapper.findAll(".device-row")[1].find(".device-remove-button").trigger("click");
		expect(confirm.last().message).toContain('"Phone"');
		expect(removeDevice).not.toHaveBeenCalled();
		confirm.last().accept?.();
		await flushPromises();
		expect(removeDevice).toHaveBeenCalledWith("d2");
		expect(wrapper.findAll(".device-row")).toHaveLength(1);
		expect(wrapper.find(".request-feedback").text()).toBe('"Phone" was removed.');
		expect(toast.add).not.toHaveBeenCalled();
		wrapper.unmount();
	});

	it("removing this device signs it out and moves to the pairing screen", async () => {
		const { wrapper, router, confirm, toast } = await mountDevices();
		await wrapper.findAll(".device-row")[0].find(".device-remove-button").trigger("click");
		expect(confirm.last().message).toContain("This is the device you are using");
		confirm.last().accept?.();
		await flushPromises();
		expect(removeDevice).toHaveBeenCalledWith("d1");
		expect(useSession().value.status).toBe("unpaired");
		expect(router.currentRoute.value.path).toBe("/pair");
		expect(toast.summaries()).toEqual(["This device was removed"]);
		wrapper.unmount();
	});

	it("renames a device in place and confirms inline", async () => {
		vi.mocked(renameDevice).mockResolvedValue({ ...phone, name: "Kitchen phone" });
		const { wrapper, toast } = await mountDevices();
		await wrapper.findAll(".device-row")[1].find(".device-rename-button").trigger("click");
		const input = wrapper.find<HTMLInputElement>(".device-rename input");
		expect(wrapper.find(".device-rename label").text()).toBe("Device name");
		expect(input.element.value).toBe("Phone");
		expect(input.attributes("maxlength")).toBe("64");
		await input.setValue("  Kitchen phone ");
		await wrapper.find(".device-rename").trigger("submit");
		await flushPromises();
		expect(renameDevice).toHaveBeenCalledWith("d2", "Kitchen phone");
		expect(wrapper.find(".device-rename").exists()).toBe(false);
		const names = wrapper.findAll(".device-row").map((row) => row.find(".device-name span").text());
		expect(names).toEqual(["Laptop", "Kitchen phone"]);
		expect(wrapper.find(".request-feedback").text()).toBe('"Kitchen phone" saved.');
		expect(toast.add).not.toHaveBeenCalled();
		wrapper.unmount();
	});

	it("keeps the session in step when this device is renamed", async () => {
		vi.mocked(renameDevice).mockResolvedValue({ ...laptop, name: "Work laptop" });
		const { wrapper } = await mountDevices();
		await wrapper.findAll(".device-row")[0].find(".device-rename-button").trigger("click");
		await wrapper.find(".device-rename input").setValue("Work laptop");
		await wrapper.find(".device-rename input").trigger("keydown", { key: "Enter" });
		await flushPromises();
		expect(renameDevice).toHaveBeenCalledWith("d1", "Work laptop");
		const session = useSession().value;
		expect(session.status === "paired" && session.device.name).toBe("Work laptop");
		wrapper.unmount();
	});

	it("cancels renaming with the button or Escape without saving", async () => {
		const { wrapper } = await mountDevices();
		const row = () => wrapper.findAll(".device-row")[1];
		await row().find(".device-rename-button").trigger("click");
		await row().find(".device-rename input").setValue("Something else");
		const cancel = row()
			.findAll("button")
			.find((button) => button.text() === "Cancel")!;
		await cancel.trigger("click");
		expect(row().find(".device-rename").exists()).toBe(false);
		expect(row().find(".device-name span").text()).toBe("Phone");

		await row().find(".device-rename-button").trigger("click");
		expect(row().find<HTMLInputElement>(".device-rename input").element.value).toBe("Phone");
		await row().find(".device-rename input").setValue("Other");
		await row().find(".device-rename input").trigger("keydown", { key: "Escape" });
		expect(row().find(".device-rename").exists()).toBe(false);
		expect(row().find(".device-name span").text()).toBe("Phone");
		expect(renameDevice).not.toHaveBeenCalled();
		wrapper.unmount();
	});

	it("shows a rejected name under the field and keeps editing", async () => {
		const message = "device name is required and must be at most 64 characters";
		vi.mocked(renameDevice).mockRejectedValue(new ApiError(message, 422, { name: message }));
		const { wrapper } = await mountDevices();
		await wrapper.findAll(".device-row")[1].find(".device-rename-button").trigger("click");
		const input = wrapper.find(".device-rename input");
		await input.setValue("x".repeat(64));
		await input.trigger("keydown", { key: "Enter" });
		await flushPromises();
		const error = wrapper.find(".device-rename .field-error");
		expect(error.text()).toBe(message);
		expect(input.attributes("aria-invalid")).toBe("true");
		expect(input.attributes("aria-describedby")).toBe(error.attributes("id"));
		expect(wrapper.find(".request-feedback").exists()).toBe(false);

		await input.setValue("   ");
		await input.trigger("keydown", { key: "Enter" });
		expect(renameDevice).toHaveBeenCalledTimes(1);
		expect(wrapper.find(".device-rename .field-error").text()).toBe("Device name is required");
		wrapper.unmount();
	});

	it("shows a pairing code with its time left and reports the device that used it", async () => {
		vi.useFakeTimers({
			toFake: ["setTimeout", "clearTimeout", "setInterval", "clearInterval", "Date"],
		});
		vi.setSystemTime(new Date("2026-09-26T10:00:00Z"));
		vi.mocked(createPairingCode).mockResolvedValue({
			code: "K7QM-3XRD",
			expiresAt: "2026-09-26T10:10:00Z",
		});
		const { wrapper } = await mountDevices();
		const pairButton = wrapper
			.findAll("button")
			.find((button) => button.text() === "Pair a new device")!;
		await pairButton.trigger("click");
		await flushPromises();
		expect(wrapper.find(".pairing-code").text()).toBe("K7QM-3XRD");
		expect(wrapper.find(".pairing-code-expiry").text()).toBe("Valid for 10:00");
		expect((wrapper.find(".pairing-link input").element as HTMLInputElement).value).toMatch(
			/\/pair\?code=K7QM3XRD$/,
		);

		vi.mocked(listDevices).mockResolvedValue([laptop, phone, tablet]);
		await vi.advanceTimersByTimeAsync(3000);
		await flushPromises();
		expect(wrapper.find(".pairing-code").exists()).toBe(false);
		expect(wrapper.find(".request-feedback").text()).toBe('"Tablet" is now paired.');
		expect(wrapper.findAll(".device-row")).toHaveLength(3);
		wrapper.unmount();
	});

	it("offers a new code once the shown one expires", async () => {
		vi.useFakeTimers({
			toFake: ["setTimeout", "clearTimeout", "setInterval", "clearInterval", "Date"],
		});
		vi.setSystemTime(new Date("2026-09-26T10:00:00Z"));
		vi.mocked(createPairingCode).mockResolvedValue({
			code: "K7QM-3XRD",
			expiresAt: "2026-09-26T10:00:05Z",
		});
		const { wrapper } = await mountDevices();
		await wrapper
			.findAll("button")
			.find((button) => button.text() === "Pair a new device")!
			.trigger("click");
		await flushPromises();
		await vi.advanceTimersByTimeAsync(6000);
		expect(wrapper.text()).toContain("The code has expired.");
		const renew = wrapper.findAll("button").find((button) => button.text() === "New code")!;
		await renew.trigger("click");
		await flushPromises();
		expect(createPairingCode).toHaveBeenCalledTimes(2);
		wrapper.unmount();
	});

	it("explains that access control is off instead of listing devices", async () => {
		setSession({ status: "open" });
		const { wrapper } = await mountDevices();
		expect(wrapper.text()).toContain("Access control is disabled");
		expect(listDevices).not.toHaveBeenCalled();
		wrapper.unmount();
	});
});
