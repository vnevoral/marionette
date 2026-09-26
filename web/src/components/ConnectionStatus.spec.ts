import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import ConnectionStatus from "@/components/ConnectionStatus.vue";
import { FakeEventSource } from "@/test/fakeEventSource";

describe("ConnectionStatus", () => {
	beforeEach(() => {
		FakeEventSource.reset();
		vi.stubGlobal("EventSource", FakeEventSource);
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it("keeps the stream open and mirrors its state", async () => {
		const wrapper = mount(ConnectionStatus);
		expect(FakeEventSource.instances).toHaveLength(1);
		expect(wrapper.text()).toBe("Reconnecting");
		expect(wrapper.classes()).toContain("connection-disconnected");
		expect(wrapper.attributes("role")).toBe("status");

		FakeEventSource.last().open();
		await wrapper.vm.$nextTick();
		expect(wrapper.text()).toBe("Live");
		expect(wrapper.classes()).toContain("connection-connected");

		FakeEventSource.last().fail();
		await wrapper.vm.$nextTick();
		expect(wrapper.text()).toBe("Reconnecting");

		wrapper.unmount();
		expect(FakeEventSource.last().closed).toBe(true);
	});
});
