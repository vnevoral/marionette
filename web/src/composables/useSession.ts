import { readonly, ref } from "vue";
import { getSession, type Session } from "@/api";

// Access state of this browser, shared by the router guard, the app shell
// and the pairing and devices views (FR-50..FR-55). "unknown" until the
// first successful GET /api/session.
export type SessionState = Session | { status: "unknown" };

const state = ref<SessionState>({ status: "unknown" });

/** Asks the server again and remembers the answer. Network errors propagate
 * and keep the previous state. */
export async function loadSession(): Promise<SessionState> {
	state.value = await getSession();
	return state.value;
}

/** Returns the known state, asking the server only the first time. */
export async function ensureSession(): Promise<SessionState> {
	if (state.value.status !== "unknown") return state.value;
	return loadSession();
}

export function setSession(next: SessionState) {
	state.value = next;
}

/** This browser lost its pairing (401 from the API, or it removed itself). */
export function markUnpaired() {
	state.value = { status: "unpaired", bootstrap: false };
}

export function useSession() {
	return readonly(state);
}
