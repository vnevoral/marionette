<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import type { StatusChange } from "@/api";
import InlineError from "@/components/InlineError.vue";
import StatusBadge from "@/components/StatusBadge.vue";
import { formatDate, transitionDuration } from "@/ui/format";
import { EMPTY, LOADING, statusPresentation } from "@/ui/vocabulary";

// Status transitions newest first. The current (open) transition shows the
// time elapsed so far (FR-17) and is refreshed once a minute.
const props = withDefaults(
	defineProps<{ changes: StatusChange[]; loading: boolean; error?: string; now?: number }>(),
	{ error: "", now: undefined },
);

const clock = ref(Date.now());
let timer: ReturnType<typeof setInterval> | undefined;

function currentTime() {
	return props.now ?? clock.value;
}

onMounted(() => {
	timer = setInterval(() => (clock.value = Date.now()), 60_000);
});

onBeforeUnmount(() => clearInterval(timer));
</script>

<template>
	<InlineError v-if="error" :message="error" />
	<p v-else-if="loading" class="section-loading" role="status">{{ LOADING.history }}</p>
	<p v-else-if="!changes.length" class="empty-copy">{{ EMPTY.history }}</p>
	<ol v-else class="timeline">
		<li
			v-for="(change, index) in changes"
			:key="`${change.startedAt}-${index}`"
			class="timeline-row"
		>
			<StatusBadge v-bind="statusPresentation(change.state)" />
			<p>
				{{ formatDate(change.startedAt) }} ·
				<span :class="{ 'timeline-open': !change.endedAt }">
					{{ transitionDuration(change, currentTime()) }}
				</span>
				<span v-if="!change.endedAt" class="timeline-current">(current)</span>
			</p>
		</li>
	</ol>
</template>

<style scoped>
.section-loading,
.empty-copy {
	margin: var(--space-4) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
.timeline {
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
	margin: 0;
	padding: 0;
	list-style: none;
}
.timeline-row {
	display: flex;
	align-items: flex-start;
	gap: var(--space-3);
	padding: var(--space-3) 0;
	border-top: 1px solid var(--color-border);
}
.timeline-row p {
	margin: var(--space-1) 0 0;
	color: var(--color-muted);
	font-size: 0.9rem;
}
.timeline-current {
	margin-left: var(--space-1);
	color: var(--color-accent-strong);
}
</style>
