import {
	Activity,
	ArrowLeft,
	ArrowUpRight,
	Ban,
	Camera,
	Check,
	ChevronDown,
	CircleAlert,
	CircleCheck,
	CircleMinus,
	CircleQuestionMark,
	CircleX,
	Clock,
	Cloud,
	Copy,
	Cpu,
	Database,
	Ghost,
	Globe,
	HardDrive,
	Heart,
	Hourglass,
	House,
	Inbox,
	LayoutGrid,
	Lightbulb,
	Link,
	LoaderCircle,
	LockOpen,
	Monitor,
	Network,
	Pencil,
	Play,
	Plus,
	Power,
	Printer,
	RefreshCw,
	Router,
	Server,
	Settings,
	Shield,
	Trash2,
	TriangleAlert,
	Tv,
	Wifi,
	X,
	Zap,
	type LucideIcon,
} from "@lucide/vue";

// Icons are Lucide components looked up by name (ADR-0014). Card icons are
// stored by name in the card's `icon` field and must satisfy the server's
// rule (lowercase letters and digits separated by dashes, ADR-0004); only
// CARD_ICON_OPTIONS are offered by the editor.
export const CARD_ICONS = {
	monitor: { label: "Monitor", component: Monitor },
	house: { label: "House", component: House },
	server: { label: "Server", component: Server },
	cloud: { label: "Cloud", component: Cloud },
	database: { label: "Database", component: Database },
	globe: { label: "Globe", component: Globe },
	zap: { label: "Bolt", component: Zap },
	settings: { label: "Settings", component: Settings },
	shield: { label: "Shield", component: Shield },
	heart: { label: "Heart", component: Heart },
	router: { label: "Router", component: Router },
	printer: { label: "Printer", component: Printer },
	"hard-drive": { label: "Hard drive", component: HardDrive },
	wifi: { label: "Wi-Fi", component: Wifi },
	tv: { label: "TV", component: Tv },
	lightbulb: { label: "Light", component: Lightbulb },
	camera: { label: "Camera", component: Camera },
	power: { label: "Power", component: Power },
	network: { label: "Network", component: Network },
	cpu: { label: "Processor", component: Cpu },
} as const satisfies Record<string, { label: string; component: LucideIcon }>;

export type CardIconName = keyof typeof CARD_ICONS;

export const CARD_ICON_OPTIONS = Object.entries(CARD_ICONS).map(([value, { label }]) => ({
	label,
	value: value as CardIconName,
}));

export const DEFAULT_CARD_ICON: CardIconName = "monitor";

/** Icons of the UI itself: statuses, buttons and navigation. */
const UI_ICONS = {
	activity: Activity,
	"arrow-left": ArrowLeft,
	"arrow-up-right": ArrowUpRight,
	ban: Ban,
	check: Check,
	"chevron-down": ChevronDown,
	"circle-alert": CircleAlert,
	"circle-check": CircleCheck,
	"circle-minus": CircleMinus,
	"circle-question-mark": CircleQuestionMark,
	"circle-x": CircleX,
	clock: Clock,
	copy: Copy,
	ghost: Ghost,
	hourglass: Hourglass,
	inbox: Inbox,
	"layout-grid": LayoutGrid,
	link: Link,
	"loader-circle": LoaderCircle,
	"lock-open": LockOpen,
	pencil: Pencil,
	play: Play,
	plus: Plus,
	"refresh-cw": RefreshCw,
	"trash-2": Trash2,
	"triangle-alert": TriangleAlert,
	x: X,
} as const satisfies Record<string, LucideIcon>;

export type IconName = CardIconName | keyof typeof UI_ICONS;

/** Whether a stored card icon is one the editor offers. */
export function isCardIcon(name: string | undefined): name is CardIconName {
	return Boolean(name) && Object.hasOwn(CARD_ICONS, name as string);
}

/**
 * The component for an icon name: an empty name is the default card icon and
 * a name the UI does not know (for example an old `pi pi-home` value) is the
 * ghost, so the card still renders and the editor asks for a new icon.
 */
export function iconComponent(name: string | undefined): LucideIcon {
	if (!name) return CARD_ICONS[DEFAULT_CARD_ICON].component;
	if (isCardIcon(name)) return CARD_ICONS[name].component;
	if (Object.hasOwn(UI_ICONS, name)) return UI_ICONS[name as keyof typeof UI_ICONS];
	return Ghost;
}
