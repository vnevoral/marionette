import { vi } from "vitest";
import { ConfirmKey, type ConfirmOptions } from "@/ui/feedback";

// Stand-in for the confirm service: records the last require() call so a
// test can trigger accept/reject without mounting MConfirmHost.
export function fakeConfirm() {
	const require = vi.fn<(options: ConfirmOptions) => void>();
	return {
		provide: { [ConfirmKey as symbol]: { require } },
		require,
		last(): ConfirmOptions {
			const options = require.mock.calls.at(-1)?.[0];
			if (!options) throw new Error("confirm.require was not called");
			return options;
		},
	};
}
