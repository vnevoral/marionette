import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { h } from "vue";
import { createMemoryHistory, createRouter } from "vue-router";
import PrimeVue from "primevue/config";
import { createPairingCode, listDevices, removeDevice, type Device } from "@/api";
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
		expect(rows[1].find("button").attributes("aria-label")).toBe("Remove device Phone");
		wrapper.unmount();
	});

	it("removes another device after confirmation and says so inline", async () => {
		const { wrapper, confirm, toast } = await mountDevices();
		await wrapper.findAll(".device-row")[1].find("button").trigger("click");
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
		await wrapper.findAll(".device-row")[0].find("button").trigger("click");
		expect(confirm.last().message).toContain("This is the device you are using");
		confirm.last().accept?.();
		await flushPromises();
		expect(removeDevice).toHaveBeenCalledWith("d1");
		expect(useSession().value.status).toBe("unpaired");
		expect(router.currentRoute.value.path).toBe("/pair");
		expect(toast.summaries()).toEqual(["This device was removed"]);
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
