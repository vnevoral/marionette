<script setup lang="ts">
import { computed } from "vue";
import type { PendingRequest, RequestResult } from "@/types";
import { REQUEST } from "@/ui/vocabulary";

// The quiet line of a card or the detail Actions panel (UX spec §4): it shows
// `fallback` (for example "Last checked ...") and is replaced in place by an
// action outcome the badge cannot express, so the layout never jumps. Pending
// phases and quiet outcomes are only announced to assistive technology; the
// badge and the button spinner already show them.
const props = withDefaults(
	defineProps<{
		fallback?: string;
		pending?: PendingRequest | null;
		result?: RequestResult | null;
		/** Keep the line to one row (dashboard card); otherwise it wraps. */
		singleLine?: boolean;
	}>(),
	{ fallback: "", pending: null, result: null, singleLine: false },
);

const visible = computed(() => (props.result && !props.result.quiet ? props.result : null));
const announcement = computed(() =>
	props.pending ? REQUEST[props.pending.phase].label : (props.result?.message ?? ""),
);
</script>

<template>
	<p
		class="action-note"
		:class="[visible ? `action-note-${visible.tone}` : '', { 'action-note-single': singleLine }]"
		:title="singleLine && visible ? visible.message : undefined"
	>
		<template v-if="visible">
			<i
				:class="visible.tone === 'error' ? 'pi pi-exclamation-circle' : 'pi pi-check'"
				aria-hidden="true"
			/>
			<span class="action-note-text">{{ visible.message }}</span>
		</template>
		<span v-else class="action-note-text">{{ fallback }}</span>
		<span class="sr-only" role="status" aria-live="polite">{{ announcement }}</span>
	</p>
</template>

<style scoped>
.action-note {
	display: flex;
	align-items: baseline;
	gap: var(--space-2);
	min-height: 1.5em;
	margin: 0;
	color: var(--color-muted);
	font-size: 0.85rem;
	line-height: 1.5;
}
.action-note-single {
	height: 1.5em;
}
.action-note-single .action-note-text {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
.action-note-success {
	color: var(--color-success-strong);
	font-weight: var(--font-weight-medium);
}
.action-note-error {
	color: var(--color-danger-strong);
	font-weight: var(--font-weight-medium);
}
</style>
