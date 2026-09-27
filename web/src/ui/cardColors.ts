// Card colours offered by the card editor (FR-10a). The value is stored in
// the card's `color` field and must be one of the server's `CardColors`
// (internal/config/types.go, a Go test keeps the two lists equal); the shade
// is the `--card-color-<value>` token in styles/tokens.css. The palette has
// no red or green: those mean Problem and Healthy on the dashboard.
export const CARD_COLOR_OPTIONS = [
	{ label: "None", value: "" },
	{ label: "Black", value: "black" },
	{ label: "Blue", value: "blue" },
	{ label: "Teal", value: "teal" },
	{ label: "Purple", value: "purple" },
	{ label: "Pink", value: "pink" },
	{ label: "Orange", value: "orange" },
	{ label: "Yellow", value: "yellow" },
] as const;

/** The CSS colour of a stored card colour, or undefined for none or an unknown name. */
export function cardColorValue(color: string | undefined): string | undefined {
	if (!color || !CARD_COLOR_OPTIONS.some((option) => option.value === color)) return undefined;
	return `var(--card-color-${color})`;
}
