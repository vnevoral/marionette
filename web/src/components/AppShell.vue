<script setup lang="ts">
import { computed } from "vue";
import { RouterLink, useRoute } from "vue-router";
import ConnectionStatus from "@/components/ConnectionStatus.vue";
import { useSession } from "@/composables/useSession";

const route = useRoute();
const session = useSession();

// Devices appears only when access control is on and this browser is
// paired; the pairing screen shows neither navigation nor the live stream.
const navigation = computed(() => [
	{ label: "Overview", to: "/", icon: "pi pi-th-large" },
	...(session.value.status === "paired"
		? [{ label: "Devices", to: "/devices", icon: "pi pi-desktop" }]
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
					<i :class="item.icon" aria-hidden="true"></i>
					<span>{{ item.label }}</span>
				</RouterLink>
			</nav>
			<ConnectionStatus v-if="!pairing" class="connection-slot" />
		</header>
		<div class="app-content">
			<slot />
		</div>
	</div>
</template>

<style scoped>
.app-shell {
	min-height: 100vh;
	background: var(--color-canvas);
}

.app-header {
	display: flex;
	align-items: center;
	gap: var(--space-8);
	min-height: 72px;
	padding: 0 clamp(var(--space-4), 5vw, 64px);
	border-bottom: 1px solid var(--color-border);
	background: color-mix(in srgb, var(--color-surface) 92%, transparent);
}

.brand {
	display: inline-flex;
	align-items: center;
	gap: var(--space-3);
	color: var(--color-ink);
	font-family: var(--font-display);
	font-size: 1.25rem;
	font-weight: var(--font-weight-semibold);
	text-decoration: none;
}

.brand-mark {
	display: grid;
	width: 32px;
	height: 32px;
	place-items: center;
	border-radius: var(--radius-sm);
	background: var(--color-accent);
	color: #ffffff;
	font-family: var(--font-ui);
	font-size: 0.9rem;
	font-weight: var(--font-weight-semibold);
}

.primary-nav {
	display: flex;
	align-self: stretch;
	gap: var(--space-2);
}

.nav-link {
	display: inline-flex;
	align-items: center;
	gap: var(--space-2);
	padding: 0 var(--space-3);
	border-bottom: 3px solid transparent;
	color: var(--color-muted);
	font-size: 0.9rem;
	font-weight: var(--font-weight-medium);
	text-decoration: none;
}

.nav-link:hover,
.nav-link.router-link-active {
	border-bottom-color: var(--color-accent);
	color: var(--color-ink);
}

.connection-slot {
	margin-left: auto;
}

.app-content {
	min-width: 0;
}

@media (max-width: 48rem) {
	.app-header {
		flex-wrap: wrap;
		gap: 0;
		padding: var(--space-3) var(--space-4) 0;
	}

	.brand {
		padding-bottom: var(--space-3);
	}

	.connection-slot {
		padding-bottom: var(--space-3);
	}

	.primary-nav {
		order: 3;
		width: 100%;
		height: 44px;
	}

	.nav-link {
		padding: 0 var(--space-2);
		font-size: 0.82rem;
	}
}
</style>
