import { vi, type Mock } from "vitest";
import { ToastKey, type ToastOptions } from "@/ui/feedback";

// Stand-in for the toast service: records added messages so a test can
// assert on them without mounting MToastHost.
export function fakeToast(): {
	provide: Record<symbol, unknown>;
	add: Mock<(message: ToastOptions) => void>;
	summaries(): string[];
} {
	const add = vi.fn<(message: ToastOptions) => void>();
	return {
		provide: { [ToastKey as symbol]: { add, remove: vi.fn() } },
		add,
		summaries(): string[] {
			return add.mock.calls.map(([message]) => message.summary);
		},
	};
}
