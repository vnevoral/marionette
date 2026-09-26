import { definePreset } from "@primevue/themes";
import Aura from "@primevue/themes/aura";

// Marionette's PrimeVue preset (ADR-0009, UX spec §8.2): the sage/green
// palette from tokens.css expressed as Aura semantic tokens, so components
// pick the colours up natively and no CSS override needs !important.
const MarionettePreset = definePreset(Aura, {
	semantic: {
		primary: {
			50: "#eef5f0",
			100: "#dcebe1",
			200: "#bcd8c6",
			300: "#94bfa5",
			400: "#6ca383",
			500: "#4b8969",
			600: "#397254",
			700: "#2f5c45",
			800: "#284a39",
			900: "#223d30",
			950: "#11211a",
		},
		focusRing: {
			width: "3px",
			style: "solid",
			color: "rgb(47 113 129 / 35%)",
			offset: "2px",
		},
		colorScheme: {
			light: {
				surface: {
					0: "#ffffff",
					50: "#f8faf9",
					100: "#eef2f1",
					200: "#d7e0dd",
					300: "#b9c9c3",
					400: "#98a9a3",
					500: "#68746f",
					600: "#4f5b56",
					700: "#3b4642",
					800: "#26332f",
					900: "#1b2522",
					950: "#101614",
				},
				primary: {
					color: "{primary.500}",
					contrastColor: "#ffffff",
					hoverColor: "{primary.600}",
					activeColor: "{primary.700}",
				},
				highlight: {
					background: "{primary.50}",
					focusBackground: "{primary.100}",
					color: "{primary.800}",
					focusColor: "{primary.900}",
				},
				text: {
					color: "{surface.800}",
					hoverColor: "{surface.900}",
					mutedColor: "{surface.500}",
					hoverMutedColor: "{surface.600}",
				},
				content: {
					background: "{surface.50}",
					hoverBackground: "{surface.100}",
					borderColor: "{surface.200}",
					color: "{surface.800}",
					hoverColor: "{surface.900}",
				},
				formField: {
					background: "{surface.0}",
					disabledBackground: "{surface.100}",
					filledBackground: "{surface.50}",
					borderColor: "{surface.300}",
					hoverBorderColor: "{surface.400}",
					focusBorderColor: "#2f7181",
					invalidBorderColor: "#b45d56",
					color: "{surface.800}",
					disabledColor: "{surface.500}",
					placeholderColor: "{surface.500}",
					floatLabelColor: "{surface.500}",
					iconColor: "{surface.500}",
					shadow: "none",
				},
				overlay: {
					select: {
						background: "{surface.0}",
						borderColor: "{surface.200}",
						color: "{surface.800}",
					},
					popover: {
						background: "{surface.0}",
						borderColor: "{surface.200}",
						color: "{surface.800}",
					},
					modal: {
						background: "{surface.0}",
						borderColor: "{surface.200}",
						color: "{surface.800}",
					},
				},
				list: {
					option: {
						focusBackground: "{primary.50}",
						selectedBackground: "{primary.50}",
						selectedFocusBackground: "{primary.100}",
						color: "{surface.800}",
						focusColor: "{surface.900}",
						selectedColor: "{surface.900}",
						selectedFocusColor: "{surface.900}",
					},
				},
			},
		},
	},
	components: {
		button: {
			label: { fontWeight: "500" },
		},
		toggleswitch: {
			root: {
				width: "2.75rem",
				height: "1.5rem",
				borderColor: "{surface.300}",
				hoverBorderColor: "{surface.400}",
				checkedBorderColor: "{primary.color}",
				checkedHoverBorderColor: "{primary.hover.color}",
			},
			colorScheme: {
				light: {
					root: {
						background: "#d7dfdc",
						hoverBackground: "#c9d4d0",
						checkedBackground: "{primary.color}",
						checkedHoverBackground: "{primary.hover.color}",
					},
					handle: {
						background: "{surface.0}",
						hoverBackground: "{surface.0}",
						checkedBackground: "{surface.0}",
						checkedHoverBackground: "{surface.0}",
					},
				},
			},
		},
	},
});

export default MarionettePreset;
