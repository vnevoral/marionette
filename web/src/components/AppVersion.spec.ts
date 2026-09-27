import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount as mountComponent, type VueWrapper } from "@vue/test-utils";
import { getHealth } from "@/api";
import AppVersion from "@/components/AppVersion.vue";
import { useStatusEvents } from "@/composables/useStatusEvents";
import { FakeEventSource } from "@/test/fakeEventSource";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return { ...actual, getHealth: vi.fn() };
});

function health(version: string) {
	return { status: "ok", version, uptimeSec: 1 };
}

const mounted: VueWrapper[] = [];

function mount(component: typeof AppVersion) {
	const wrapper = mountComponent(component);
	mounted.push(wrapper);
	return wrapper;
}

describe("AppVersion", () => {
	let unsubscribe: () => void = () => {};

	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
		vi.mocked(getHealth).mockReset();
	});

	afterEach(() => {
		for (const wrapper of mounted.splice(0)) wrapper.unmount();
		unsubscribe();
		vi.unstubAllGlobals();
	});

	it("shows the running version", async () => {
		vi.mocked(getHealth).mockResolvedValue(health("v1.1.0"));
		const wrapper = mount(AppVersion);
		await flushPromises();
		expect(wrapper.text()).toBe("Marionette v1.1.0");
	});

	it("shows nothing when the version cannot be read", async () => {
		vi.mocked(getHealth).mockRejectedValue(new Error("offline"));
		const wrapper = mount(AppVersion);
		await flushPromises();
		expect(wrapper.find(".app-version").exists()).toBe(false);
	});

	it("reads the version again when the stream reconnects, not on the first connect", async () => {
		vi.mocked(getHealth).mockResolvedValueOnce(health("v1.0.0"));
		unsubscribe = useStatusEvents().subscribe({ onStatus() {} });
		const wrapper = mount(AppVersion);
		await flushPromises();
		FakeEventSource.last().open();
		await flushPromises();
		expect(getHealth).toHaveBeenCalledTimes(1);

		// The service restarts with a new version.
		vi.mocked(getHealth).mockResolvedValueOnce(health("v1.1.0"));
		FakeEventSource.last().fail();
		await flushPromises();
		FakeEventSource.last().open();
		await flushPromises();
		expect(getHealth).toHaveBeenCalledTimes(2);
		expect(wrapper.text()).toBe("Marionette v1.1.0");
	});
});
