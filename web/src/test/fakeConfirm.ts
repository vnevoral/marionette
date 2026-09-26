import type { ConfirmationOptions } from "primevue/confirmationoptions";
import * as useConfirmModule from "primevue/useconfirm";
import { vi } from "vitest";

// PrimeVue exports the injection key at runtime but not in its typings.
const { PrimeVueConfirmSymbol } = useConfirmModule as unknown as { PrimeVueConfirmSymbol: symbol };

// Stand-in for PrimeVue's ConfirmationService: records the last require()
// call so a test can trigger accept/reject without mounting ConfirmDialog.
export function fakeConfirm() {
	const require = vi.fn<(options: ConfirmationOptions) => void>();
	return {
		provide: { [PrimeVueConfirmSymbol]: { require, close: vi.fn() } },
		require,
		last(): ConfirmationOptions {
			const options = require.mock.calls.at(-1)?.[0];
			if (!options) throw new Error("confirm.require was not called");
			return options;
		},
	};
}
