import { useToast } from "primevue/usetoast";
import { messageVisibleMs } from "@/composables/useTransientMessage";

/**
 * Page-level notices shown as a toast (UX spec §4 "Feedback channels"): the
 * outcome of an operation that ends by leaving or replacing the page, such as
 * saving or deleting a card, so the notice survives the navigation. Feedback
 * that belongs to one card or one form field stays inline next to it.
 */
export function useNotify() {
	const toast = useToast();
	return {
		success(summary: string, detail?: string) {
			toast.add({ severity: "success", summary, detail, life: messageVisibleMs });
		},
	};
}
