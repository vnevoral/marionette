<script setup lang="ts">
import {
	ToastClose,
	ToastDescription,
	ToastProvider,
	ToastRoot,
	ToastTitle,
	ToastViewport,
} from "reka-ui";
import { useFeedbackState, useToast, type ToastTone } from "@/ui/feedback";
import MIcon from "@/ui/MIcon.vue";

// Page-level notices (useNotify, UX spec §4): top right, full width with the
// page gutter on phones. Reka UI pauses the timer while a toast is hovered or
// focused and announces it to screen readers.
const state = useFeedbackState();
const toast = useToast();

const TONE_ICONS: Record<ToastTone, string> = {
	success: "circle-check",
	info: "circle-alert",
	error: "circle-x",
};
</script>

<template>
	<ToastProvider swipe-direction="right">
		<ToastRoot
			v-for="message in state.toasts"
			:key="message.id"
			class="m-toast"
			:class="`m-toast-${message.tone}`"
			:duration="message.life"
			:default-open="true"
			@update:open="(open: boolean) => !open && toast.remove(message.id)"
		>
			<MIcon class="m-toast-icon" :name="TONE_ICONS[message.tone]" size="1.25rem" />
			<div class="m-toast-text">
				<ToastTitle class="m-toast-summary">{{ message.summary }}</ToastTitle>
				<ToastDescription v-if="message.detail" class="m-toast-detail">
					{{ message.detail }}
				</ToastDescription>
			</div>
			<ToastClose class="m-toast-close" aria-label="Close">
				<MIcon name="x" />
			</ToastClose>
		</ToastRoot>
		<ToastViewport class="m-toast-viewport" />
	</ToastProvider>
</template>

<style>
.m-toast-viewport {
	position: fixed;
	top: var(--space-4);
	right: var(--space-4);
	z-index: 60;
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
	width: min(24rem, calc(100vw - 2 * var(--space-4)));
	max-width: 100vw;
	margin: 0;
	padding: 0;
	list-style: none;
	outline: none;
}
.m-toast {
	display: flex;
	align-items: flex-start;
	gap: var(--space-3);
	padding: var(--space-4);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-surface);
	box-shadow: var(--shadow-md);
	color: var(--color-ink);
	animation: m-toast-in 160ms ease-out;
}
.m-toast[data-state="closed"] {
	animation: m-toast-out 120ms ease-in forwards;
}
.m-toast[data-swipe="move"] {
	transform: translateX(var(--reka-toast-swipe-move-x));
}
.m-toast[data-swipe="end"] {
	animation: m-toast-out 120ms ease-in forwards;
}
.m-toast-icon {
	flex: none;
	margin-top: 0.0625rem;
}
.m-toast-success .m-toast-icon,
.m-toast-success .m-toast-summary {
	color: var(--color-success-strong);
}
.m-toast-info .m-toast-icon,
.m-toast-info .m-toast-summary {
	color: var(--color-info-strong);
}
.m-toast-error .m-toast-icon,
.m-toast-error .m-toast-summary {
	color: var(--color-danger-strong);
}
.m-toast-text {
	flex: 1;
	min-width: 0;
}
.m-toast-summary {
	font-weight: var(--font-weight-semibold);
	line-height: 1.4;
}
.m-toast-detail {
	margin-top: 0.125rem;
	color: var(--color-muted-strong);
	line-height: 1.5;
	overflow-wrap: anywhere;
}
.m-toast-close {
	display: inline-flex;
	flex: none;
	align-items: center;
	justify-content: center;
	width: 1.75rem;
	height: 1.75rem;
	margin: -0.25rem -0.25rem 0 0;
	border: 0;
	border-radius: var(--radius-sm);
	background: transparent;
	color: var(--color-muted);
	cursor: pointer;
}
.m-toast-close:hover {
	background: var(--color-surface-sunken);
	color: var(--color-ink);
}
@keyframes m-toast-in {
	from {
		opacity: 0;
		transform: translateY(-0.5rem);
	}
}
@keyframes m-toast-out {
	to {
		opacity: 0;
	}
}
@media (prefers-reduced-motion: reduce) {
	.m-toast,
	.m-toast[data-state="closed"] {
		animation: none;
	}
}
</style>
