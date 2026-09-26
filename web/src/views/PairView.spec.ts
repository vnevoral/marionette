import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { h } from "vue";
import { createMemoryHistory, createRouter, RouterView } from "vue-router";
import PrimeVue from "primevue/config";
import { ApiError, getSession, pairDevice, type Device } from "@/api";
import { setSession, useSession } from "@/composables/useSession";
import { fakeToast } from "@/test/fakeToast";
import PairView from "@/views/PairView.vue";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return { ...actual, getSession: vi.fn(), pairDevice: vi.fn() };
});

const device: Device = {
	id: "d1",
	name: "Laptop",
	pairedAt: "2026-09-26T10:00:00Z",
	lastSeenAt: "2026-09-26T10:00:00Z",
	current: true,
};

async function mountPair(path: string) {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{ path: "/", component: { render: () => h("div", "overview") } },
			{ path: "/cards/:id", component: { render: () => h("div", "detail") } },
			{ path: "/pair", name: "pair", component: PairView, meta: { pairing: true } },
		],
	});
	await router.push(path);
	await router.isReady();
	const toast = fakeToast();
	const wrapper = mount(
		{ render: () => h(RouterView) },
		{ global: { plugins: [router, PrimeVue], provide: toast.provide } },
	);
	await flushPromises();
	return { wrapper, router, toast };
}

function inputs(wrapper: Awaited<ReturnType<typeof mountPair>>["wrapper"]) {
	const [code, name] = wrapper.findAll("input");
	return { code, name };
}

describe("PairView", () => {
	beforeEach(() => {
		setSession({ status: "unknown" });
		vi.mocked(getSession).mockReset().mockResolvedValue({ status: "unpaired", bootstrap: false });
		vi.mocked(pairDevice).mockReset();
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it("shows the service log command only while no device is paired", async () => {
		vi.mocked(getSession).mockResolvedValue({ status: "unpaired", bootstrap: true });
		const first = await mountPair("/pair");
		expect(first.wrapper.find(".pair-command").text()).toBe(
			'journalctl -u marionette | grep "pairing code"',
		);
		first.wrapper.unmount();

		vi.mocked(getSession).mockResolvedValue({ status: "unpaired", bootstrap: false });
		const later = await mountPair("/pair");
		expect(later.wrapper.find(".pair-command").exists()).toBe(false);
		expect(later.wrapper.text()).toContain("Devices → Pair a new device");
		later.wrapper.unmount();
	});

	it("groups the typed code, fills it from the link and suggests a device name", async () => {
		vi.stubGlobal("navigator", {
			userAgent: "Mozilla/5.0 (Android 15; Mobile; rv:140.0) Gecko/140.0 Firefox/140.0",
		});
		const { wrapper } = await mountPair("/pair?code=k7qm3xrd");
		const { code, name } = inputs(wrapper);
		expect((code.element as HTMLInputElement).value).toBe("K7QM-3XRD");
		expect((name.element as HTMLInputElement).value).toBe("Firefox on Android");
		await code.setValue("ab-c d1234x9");
		expect((code.element as HTMLInputElement).value).toBe("ABCD-1234");
		wrapper.unmount();
	});

	it("pairs, remembers the device and returns to where the user was going", async () => {
		vi.mocked(pairDevice).mockResolvedValue(device);
		const { wrapper, router, toast } = await mountPair("/pair?next=/cards/printer");
		// After pairing the server reports this browser as paired.
		vi.mocked(getSession).mockResolvedValue({ status: "paired", device, expiryDays: 60 });
		const { code, name } = inputs(wrapper);
		await code.setValue("K7QM3XRD");
		await name.setValue(" Laptop ");
		await wrapper.find("form").trigger("submit");
		await flushPromises();
		expect(pairDevice).toHaveBeenCalledWith("K7QM-3XRD", "Laptop");
		expect(useSession().value.status).toBe("paired");
		expect(toast.summaries()).toEqual(["Device paired"]);
		expect(router.currentRoute.value.path).toBe("/cards/printer");
		wrapper.unmount();
	});

	it("never leaves the app after pairing", async () => {
		vi.mocked(pairDevice).mockResolvedValue(device);
		const { wrapper, router } = await mountPair("/pair?code=K7QM3XRD&next=//evil.example");
		await wrapper.find("form").trigger("submit");
		await flushPromises();
		expect(router.currentRoute.value.fullPath).toBe("/");
		wrapper.unmount();
	});

	it("shows one message for a wrong or expired code and field errors for the name", async () => {
		vi.mocked(pairDevice).mockRejectedValueOnce(
			new ApiError("the pairing code is invalid or has expired", 400),
		);
		const { wrapper, router, toast } = await mountPair("/pair?code=ZZZZZZZZ");
		await wrapper.find("form").trigger("submit");
		await flushPromises();
		expect(wrapper.find(".inline-error").text()).toBe(
			"The pairing code is invalid or has expired.",
		);
		expect(router.currentRoute.value.path).toBe("/pair");
		expect(toast.add).not.toHaveBeenCalled();

		vi.mocked(pairDevice).mockRejectedValueOnce(
			new ApiError("device name is required", 422, { name: "device name is required" }),
		);
		await inputs(wrapper).name.setValue("x");
		await wrapper.find("form").trigger("submit");
		await flushPromises();
		expect(wrapper.find(".field-error").text()).toBe("device name is required");

		await inputs(wrapper).name.setValue("  ");
		await wrapper.find("form").trigger("submit");
		await flushPromises();
		expect(wrapper.find(".field-error").text()).toBe("Device name is required");
		expect(pairDevice).toHaveBeenCalledTimes(2);
		wrapper.unmount();
	});
});
