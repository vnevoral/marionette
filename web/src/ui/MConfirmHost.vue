<script setup lang="ts">
import { computed } from "vue";
import {
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogOverlay,
	AlertDialogPortal,
	AlertDialogRoot,
	AlertDialogTitle,
} from "reka-ui";
import { useFeedbackState } from "@/ui/feedback";
import MButton from "@/ui/MButton.vue";
import MIcon from "@/ui/MIcon.vue";

// The shared confirmation dialog (useConfirm). Reka UI gives the initial
// focus to the cancel button, so Enter never confirms by accident.
const state = useFeedbackState();
const question = computed(() => state.confirm);

function close() {
	state.confirm = null;
}

function accept() {
	const current = state.confirm;
	close();
	current?.accept?.();
}

function onOpenChange(open: boolean) {
	if (open) return;
	// Escape or the cancel button.
	const current = state.confirm;
	close();
	current?.reject?.();
}
</script>

<template>
	<AlertDialogRoot :open="question !== null" @update:open="onOpenChange">
		<AlertDialogPortal>
			<AlertDialogOverlay class="m-confirm-overlay" />
			<AlertDialogContent v-if="question" class="m-confirm">
				<div class="m-confirm-head">
					<span class="m-confirm-icon" :class="{ 'm-confirm-icon-danger': question.danger }">
						<MIcon name="triangle-alert" size="1.25rem" />
					</span>
					<div class="m-confirm-text">
						<AlertDialogTitle class="m-confirm-title">{{ question.header }}</AlertDialogTitle>
						<AlertDialogDescription class="m-confirm-message">
							{{ question.message }}
						</AlertDialogDescription>
					</div>
				</div>
				<div class="m-confirm-actions">
					<AlertDialogCancel as-child>
						<MButton :label="question.rejectLabel" />
					</AlertDialogCancel>
					<!-- Not AlertDialogAction: closing through it would also reject. -->
					<MButton
						:label="question.acceptLabel"
						:variant="question.danger ? 'danger' : 'primary'"
						@click="accept"
					/>
				</div>
			</AlertDialogContent>
		</AlertDialogPortal>
	</AlertDialogRoot>
</template>

<style>
.m-confirm-overlay {
	position: fixed;
	inset: 0;
	z-index: 50;
	background: rgb(16 40 30 / 35%);
	animation: m-fade-in 120ms ease-out;
}
.m-confirm {
	position: fixed;
	top: 50%;
	left: 50%;
	z-index: 51;
	width: min(28rem, calc(100vw - 2 * var(--space-4)));
	padding: var(--space-6);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	background: var(--color-surface);
	box-shadow: var(--shadow-lg);
	color: var(--color-ink);
	transform: translate(-50%, -50%);
	animation: m-pop-in 140ms ease-out;
}
.m-confirm:focus {
	outline: none;
}
.m-confirm-head {
	display: flex;
	align-items: flex-start;
	gap: var(--space-4);
}
.m-confirm-icon {
	display: inline-flex;
	flex: none;
	align-items: center;
	justify-content: center;
	width: 2.5rem;
	height: 2.5rem;
	border-radius: 999px;
	background: var(--color-warning-soft);
	color: var(--color-warning-strong);
}
.m-confirm-icon-danger {
	background: var(--color-danger-soft);
	color: var(--color-danger-strong);
}
.m-confirm-text {
	min-width: 0;
}
.m-confirm-title {
	margin: 0;
	font-size: 1.125rem;
	font-weight: var(--font-weight-semibold);
	line-height: 1.4;
}
.m-confirm-message {
	margin: var(--space-1) 0 0;
	color: var(--color-muted-strong);
	line-height: 1.5;
	overflow-wrap: anywhere;
}
.m-confirm-actions {
	display: flex;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: var(--space-3);
	margin-top: var(--space-6);
}
@media (max-width: 24rem) {
	.m-confirm-actions > * {
		flex: 1 1 100%;
	}
}
@keyframes m-fade-in {
	from {
		opacity: 0;
	}
}
@keyframes m-pop-in {
	from {
		opacity: 0;
		transform: translate(-50%, -48%) scale(0.98);
	}
}
@media (prefers-reduced-motion: reduce) {
	.m-confirm-overlay,
	.m-confirm {
		animation: none;
	}
}
</style>
