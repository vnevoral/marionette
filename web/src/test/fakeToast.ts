import type { ToastMessageOptions } from "primevue/toast";
import * as useToastModule from "primevue/usetoast";
import { vi } from "vitest";

// PrimeVue exports the injection key at runtime but not in its typings.
const { PrimeVueToastSymbol } = useToastModule as unknown as { PrimeVueToastSymbol: symbol };

// Stand-in for PrimeVue's ToastService: records added messages so a test can
// assert on them without mounting the Toast component.
export function fakeToast() {
	const add = vi.fn<(message: ToastMessageOptions) => void>();
	return {
		provide: {
			[PrimeVueToastSymbol]: {
				add,
				remove: vi.fn(),
				removeGroup: vi.fn(),
				removeAllGroups: vi.fn(),
			},
		},
		add,
		summaries(): string[] {
			return add.mock.calls.map(([message]) => String(message.summary));
		},
	};
}
