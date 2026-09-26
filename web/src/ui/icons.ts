// Icon choices offered by the card editor. Values are PrimeIcons classes and
// must satisfy the server's `icon` rule (`pi pi-<name>`, ADR-0004).
export const CARD_ICON_OPTIONS = [
	{ label: "Desktop", value: "pi pi-desktop" },
	{ label: "Home", value: "pi pi-home" },
	{ label: "Server", value: "pi pi-server" },
	{ label: "Cloud", value: "pi pi-cloud" },
	{ label: "Database", value: "pi pi-database" },
	{ label: "Globe", value: "pi pi-globe" },
	{ label: "Bolt", value: "pi pi-bolt" },
	{ label: "Cog", value: "pi pi-cog" },
	{ label: "Shield", value: "pi pi-shield" },
	{ label: "Heart", value: "pi pi-heart" },
] as const;

export const DEFAULT_CARD_ICON = CARD_ICON_OPTIONS[0].value;
