import { onBeforeUnmount, onMounted, type Ref } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { useConfirm } from "primevue/useconfirm";

// Guards a dirty form (UX spec §7.3): in-app navigation asks through the
// shared ConfirmDialog, a page unload through the browser's own prompt.
// Must be called during setup of a component rendered by RouterView.
export function useUnsavedChangesGuard(isDirty: Ref<boolean>, bypass: () => boolean = () => false) {
	const confirm = useConfirm();

	function handleBeforeUnload(event: BeforeUnloadEvent) {
		if (!isDirty.value || bypass()) return;
		event.preventDefault();
		event.returnValue = "";
	}

	function confirmDiscard(): Promise<boolean> {
		return new Promise((resolve) => {
			confirm.require({
				header: "Discard unsaved changes?",
				message: "Your edits to this card have not been saved.",
				icon: "pi pi-exclamation-triangle",
				acceptLabel: "Discard changes",
				rejectLabel: "Keep editing",
				acceptProps: { severity: "danger" },
				rejectProps: { severity: "secondary", outlined: true },
				defaultFocus: "reject",
				accept: () => resolve(true),
				reject: () => resolve(false),
				onHide: () => resolve(false),
			});
		});
	}

	onMounted(() => window.addEventListener("beforeunload", handleBeforeUnload));
	onBeforeUnmount(() => window.removeEventListener("beforeunload", handleBeforeUnload));
	onBeforeRouteLeave(() => {
		if (!isDirty.value || bypass()) return true;
		return confirmDiscard();
	});
}
