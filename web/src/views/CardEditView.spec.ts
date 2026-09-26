import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { h } from "vue";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import { createMemoryHistory, createRouter, RouterView, type Router } from "vue-router";
import PrimeVue from "primevue/config";
import { ApiError, createCard, getCard, updateCard, type ActionCard } from "@/api";
import CardEditView from "@/views/CardEditView.vue";
import { fakeConfirm } from "@/test/fakeConfirm";
import { fakeToast } from "@/test/fakeToast";

vi.mock("@/api", async (importOriginal) => {
	const actual = await importOriginal<typeof import("@/api")>();
	return { ...actual, getCard: vi.fn(), createCard: vi.fn(), updateCard: vi.fn() };
});

const printer: ActionCard = {
	id: "printer",
	name: "Printer",
	description: "",
	icon: "pi pi-desktop",
	primary: {
		command: "wake",
		args: ["--now"],
		env: { TZ: "UTC" },
		timeoutSec: 5,
		rule: { type: "exit_code" },
	},
};

const Blank = { render: () => h("div") };

function inputInLabel(wrapper: VueWrapper, label: string) {
	const node = wrapper.findAll("label").find((candidate) => candidate.text().startsWith(label));
	if (!node) throw new Error(`label ${label} not rendered`);
	return node.find("input, textarea");
}

function commandLine(wrapper: VueWrapper) {
	return wrapper.find<HTMLInputElement>("input.command-line");
}

async function mountEdit(path: string) {
	const router: Router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{ path: "/", component: Blank },
			{ path: "/cards/new/edit", name: "card-new", component: CardEditView },
			{ path: "/cards/:id", name: "card-detail", component: Blank },
			{ path: "/cards/:id/edit", name: "card-edit", component: CardEditView },
		],
	});
	await router.push(path);
	await router.isReady();
	const confirm = fakeConfirm();
	const toast = fakeToast();
	// Rendered through RouterView so onBeforeRouteLeave is attached to the route.
	const wrapper = mount(
		{ render: () => h(RouterView) },
		{ global: { plugins: [router, PrimeVue], provide: { ...confirm.provide, ...toast.provide } } },
	);
	await flushPromises();
	return { wrapper, router, confirm, toast };
}

describe("CardEditView", () => {
	beforeEach(() => {
		vi.mocked(getCard).mockReset().mockResolvedValue(printer);
		vi.mocked(updateCard).mockReset();
		vi.mocked(createCard).mockReset();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it("shows 422 field messages beside the inputs and keeps the values", async () => {
		vi.mocked(updateCard).mockRejectedValue(
			new ApiError("validation failed: name: too long; primary.command: bad", 422, {
				name: "name is too long",
				"primary.command": "action command is required",
				"primary.env.TZ": "environment variable name is invalid",
				id: "immutable",
			}),
		);
		const { wrapper } = await mountEdit("/cards/printer/edit");
		await inputInLabel(wrapper, "Name").setValue("Printer renamed");
		await wrapper
			.findAll("button")
			.find((b) => b.text().includes("Save card"))!
			.trigger("click");
		await flushPromises();

		const errors = wrapper.findAll(".field-error").map((node) => node.text());
		expect(errors).toContain("name is too long");
		expect(errors).toContain("action command is required");
		expect(errors).toContain("environment variable name is invalid");
		expect(wrapper.find(".request-feedback-error").text()).toContain("validation failed");
		expect(wrapper.find(".request-feedback-error").text()).toContain("id: immutable");
		expect((inputInLabel(wrapper, "Name").element as HTMLInputElement).value).toBe(
			"Printer renamed",
		);
		expect(commandLine(wrapper).element.value).toBe("wake --now");
		wrapper.unmount();
	});

	it("asks before leaving a dirty form through the confirm dialog", async () => {
		const { wrapper, router, confirm } = await mountEdit("/cards/printer/edit");
		await inputInLabel(wrapper, "Name").setValue("Changed");

		const blocked = router.push("/");
		await flushPromises();
		expect(confirm.require).toHaveBeenCalledTimes(1);
		expect(confirm.last().header).toBe("Discard unsaved changes?");
		confirm.last().reject?.();
		await blocked;
		expect(router.currentRoute.value.path).toBe("/cards/printer/edit");

		const allowed = router.push("/");
		await flushPromises();
		confirm.last().accept?.();
		await allowed;
		expect(router.currentRoute.value.path).toBe("/");
		wrapper.unmount();
	});

	it("does not ask when the form is clean", async () => {
		const { wrapper, router, confirm } = await mountEdit("/cards/printer/edit");
		await router.push("/");
		expect(confirm.require).not.toHaveBeenCalled();
		expect(router.currentRoute.value.path).toBe("/");
		wrapper.unmount();
	});

	it("keeps a new card in the edit context after saving", async () => {
		vi.mocked(createCard).mockImplementation(async (card) => ({ ...card, id: "card-1" }));
		const { wrapper, router, confirm, toast } = await mountEdit("/cards/new/edit");
		expect(wrapper.find("h1").text()).toBe("New card");
		await inputInLabel(wrapper, "Name").setValue("Lamp");
		await commandLine(wrapper).setValue("switch --on 'Living room'");
		await wrapper
			.findAll("button")
			.find((b) => b.text().includes("Save card"))!
			.trigger("click");
		await flushPromises();

		expect(createCard).toHaveBeenCalledTimes(1);
		expect(vi.mocked(createCard).mock.calls[0][0]).toMatchObject({
			id: "",
			name: "Lamp",
			primary: { command: "switch", args: ["--on", "Living room"] },
		});
		expect(router.currentRoute.value.path).toBe("/cards/card-1/edit");
		expect(getCard).not.toHaveBeenCalled();
		expect((inputInLabel(wrapper, "Name").element as HTMLInputElement).value).toBe("Lamp");
		// The notice is a toast so it survives the move to the edit route.
		expect(toast.summaries()).toEqual(["Card saved"]);
		expect(toast.add.mock.calls[0][0]).toMatchObject({
			severity: "success",
			detail: "Lamp",
			life: 4000,
		});
		expect(wrapper.find("h1").text()).toBe("Edit card");
		expect(confirm.require).not.toHaveBeenCalled();
		wrapper.unmount();
	});

	it("saves a status action typed only as a command line", async () => {
		vi.mocked(createCard).mockImplementation(async (card) => ({ ...card, id: "card-2" }));
		const { wrapper } = await mountEdit("/cards/new/edit");
		await inputInLabel(wrapper, "Name").setValue("PC");
		await commandLine(wrapper).setValue("/usr/bin/wakeonlan AA:BB:CC:DD:EE:FF");
		await wrapper.find("#status-enabled").setValue(true);
		await wrapper.findAll("input.command-line")[1].setValue("/usr/bin/ping -c 1 -W 2 192.168.1.10");
		await wrapper
			.findAll("button")
			.find((b) => b.text().includes("Save card"))!
			.trigger("click");
		await flushPromises();

		expect(vi.mocked(createCard).mock.calls[0][0]).toMatchObject({
			primary: { command: "/usr/bin/wakeonlan", args: ["AA:BB:CC:DD:EE:FF"] },
			status: {
				command: "/usr/bin/ping",
				args: ["-c", "1", "-W", "2", "192.168.1.10"],
				timeoutSec: 30,
			},
		});
		wrapper.unmount();
	});

	it("does not save a command line that needs a shell", async () => {
		const { wrapper } = await mountEdit("/cards/printer/edit");
		await commandLine(wrapper).setValue("wake --now | tee log");
		await wrapper
			.findAll("button")
			.find((b) => b.text().includes("Save card"))!
			.trigger("click");
		await flushPromises();

		expect(updateCard).not.toHaveBeenCalled();
		expect(wrapper.find(".field-error").text()).toContain('"|" needs a shell');
		wrapper.unmount();
	});
});
