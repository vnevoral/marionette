import { inject, reactive, type App, type InjectionKey } from "vue";

// Page-level feedback shared by the whole app (ADR-0014): toasts and one
// confirmation question at a time. main.ts installs the service; MToastHost
// and MConfirmHost in App.vue render its state; tests provide a fake.

// Toasts are success notices only (UX spec §4); errors stay inline.
export type ToastTone = "success";

export interface ToastOptions {
	tone: ToastTone;
	summary: string;
	detail?: string;
	/** Milliseconds before the toast closes on its own. */
	life: number;
}

export interface ToastMessage extends ToastOptions {
	id: number;
}

export interface ConfirmOptions {
	header: string;
	message: string;
	acceptLabel: string;
	rejectLabel: string;
	/** A destructive question draws the accept button in the danger variant. */
	danger?: boolean;
	accept?: () => void;
	reject?: () => void;
}

export interface ToastService {
	add(options: ToastOptions): void;
	remove(id: number): void;
}

export interface ConfirmService {
	require(options: ConfirmOptions): void;
}

export interface FeedbackState {
	toasts: ToastMessage[];
	confirm: ConfirmOptions | null;
}

export const ToastKey: InjectionKey<ToastService> = Symbol("toast");
export const ConfirmKey: InjectionKey<ConfirmService> = Symbol("confirm");
export const FeedbackStateKey: InjectionKey<FeedbackState> = Symbol("feedback-state");

export function createFeedback() {
	const state = reactive<FeedbackState>({ toasts: [], confirm: null });
	let nextId = 0;
	const toast: ToastService = {
		add(options) {
			state.toasts.push({ ...options, id: ++nextId });
		},
		remove(id) {
			state.toasts = state.toasts.filter((message) => message.id !== id);
		},
	};
	const confirm: ConfirmService = {
		require(options) {
			// A new question replaces an open one, which counts as rejected.
			state.confirm?.reject?.();
			state.confirm = options;
		},
	};
	return {
		state,
		toast,
		confirm,
		install(app: App) {
			app.provide(FeedbackStateKey, state);
			app.provide(ToastKey, toast);
			app.provide(ConfirmKey, confirm);
		},
	};
}

function required<T>(key: InjectionKey<T>, name: string): T {
	const service = inject(key);
	if (!service) throw new Error(`${name} is not installed (see createFeedback in main.ts)`);
	return service;
}

export const useToast = () => required(ToastKey, "useToast");
export const useConfirm = () => required(ConfirmKey, "useConfirm");
export const useFeedbackState = () => required(FeedbackStateKey, "useFeedbackState");
