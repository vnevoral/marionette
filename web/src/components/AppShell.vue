<script setup lang="ts">
import { computed } from "vue";
import { RouterLink, useRoute } from "vue-router";
import AppVersion from "@/components/AppVersion.vue";
import ConnectionStatus from "@/components/ConnectionStatus.vue";
import { useSession } from "@/composables/useSession";
import MIcon from "@/ui/MIcon.vue";

const route = useRoute();
const session = useSession();

// Devices appears only when access control is on and this browser is
// paired; the pairing screen shows neither navigation nor the live stream.
const navigation = computed(() => [
	{ label: "Overview", to: "/", icon: "layout-grid" },
	...(session.value.status === "paired"
		? [{ label: "Devices", to: "/devices", icon: "monitor" }]
		: []),
]);
const pairing = computed(() => Boolean(route.meta.pairing));
</script>

<template>
	<div class="app-shell">
		<header class="app-header">
			<RouterLink class="brand" to="/" aria-label="Marionette overview">
				<span class="brand-mark" aria-hidden="true">M</span>
				<span class="brand-name">Marionette</span>
			</RouterLink>
			<nav v-if="!pairing" class="primary-nav" aria-label="Primary navigation">
				<RouterLink v-for="item in navigation" :key="item.to" :to="item.to" class="nav-link">
					<MIcon :name="item.icon" />
					<span>{{ item.label }}</span>
				</RouterLink>
			</nav>
			<ConnectionStatus v-if="!pairing" class="connection-slot" />
		</header>
		<div class="app-content">
			<slot />
		</div>
		<footer v-if="!pairing" class="app-footer">
			<AppVersion />
		</footer>
	</div>
</template>

<style scoped>
.app-shell {
	display: flex;
	flex-direction: column;
	min-height: 100vh;
	background: var(--color-canvas);
}

/* A plain sticky bar: white, a hairline border and the navigation as pills. */
.app-header {
	position: sticky;
	top: 0;
	z-index: 10;
	display: flex;
	align-items: center;
	gap: var(--space-6);
	min-height: 64px;
	padding: 0 clamp(var(--space-4), 5vw, 64px);
	border-bottom: 1px solid var(--color-border);
	background: var(--color-surface);
}

.brand {
	display: inline-flex;
	align-items: center;
	gap: var(--space-3);
	color: var(--color-ink);
	font-family: var(--font-display);
	font-size: 1.0625rem;
	font-weight: var(--font-weight-semibold);
	letter-spacing: -0.01em;
	text-decoration: none;
}

.brand-mark {
	display: grid;
	width: 32px;
	height: 32px;
	place-items: center;
	border-radius: var(--radius-sm);
	background: linear-gradient(135deg, var(--color-accent), var(--color-accent-strong));
	box-shadow: var(--shadow-xs);
	color: var(--color-on-accent);
	font-family: var(--font-ui);
	font-size: 0.9rem;
	font-weight: var(--font-weight-semibold);
}

.primary-nav {
	display: flex;
	align-items: center;
	gap: var(--space-1);
}

.nav-link {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	min-height: 36px;
	padding: 0 var(--space-3);
	border-radius: 999px;
	color: var(--color-muted);
	font-size: 0.9rem;
	font-weight: var(--font-weight-medium);
	text-decoration: none;
	transition:
		background-color 120ms ease,
		color 120ms ease;
}

.nav-link:hover {
	background: var(--color-surface-sunken);
	color: var(--color-ink);
}

.nav-link.router-link-active {
	background: var(--color-accent-soft);
	color: var(--color-accent-strong);
}

.connection-slot {
	margin-left: auto;
}

.app-content {
	flex: 1;
	min-width: 0;
}

.app-footer {
	padding: var(--space-4) clamp(var(--space-4), 5vw, 64px) var(--space-6);
	text-align: center;
}

@media (max-width: 48rem) {
	.app-header {
		flex-wrap: wrap;
		gap: 0 var(--space-3);
		padding: var(--space-3) var(--space-4) var(--space-2);
	}

	.primary-nav {
		order: 3;
		width: 100%;
		margin-top: var(--space-2);
	}

	.nav-link {
		font-size: 0.875rem;
	}
}
</style>
