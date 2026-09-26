import { EMPTY } from "@/ui/vocabulary";

const NANOSECONDS_PER_MILLISECOND = 1_000_000;

/** Formats a Go time.Duration (nanoseconds) for humans. */
export function formatDuration(durationNanoseconds: number): string {
	const milliseconds = durationNanoseconds / NANOSECONDS_PER_MILLISECOND;
	if (milliseconds < 1000) return `${Math.round(milliseconds)} ms`;
	const seconds = milliseconds / 1000;
	if (seconds < 60) return `${seconds.toFixed(2)} s`;
	return formatElapsed(milliseconds);
}

/** Coarse wall-clock span such as "3 min" or "2 h 05 min". */
export function formatElapsed(milliseconds: number): string {
	const totalMinutes = Math.floor(Math.max(milliseconds, 0) / 60_000);
	if (totalMinutes < 1) return "under a minute";
	const hours = Math.floor(totalMinutes / 60);
	const minutes = totalMinutes % 60;
	if (hours < 1) return `${minutes} min`;
	const days = Math.floor(hours / 24);
	if (days >= 1) return `${days} d ${hours % 24} h`;
	return `${hours} h ${String(minutes).padStart(2, "0")} min`;
}

export function formatDate(value?: string): string {
	if (!value) return "Not available";
	return new Date(value).toLocaleString();
}

/** "Last checked <date>" or "Not checked yet" (UX spec §5.2). */
export function lastCheckedLabel(checkedAt?: string): string {
	return checkedAt ? `Last checked ${formatDate(checkedAt)}` : EMPTY.notChecked;
}

/** Elapsed time between an ISO timestamp and `now` (defaults to the current time). */
export function elapsedSince(startedAt: string, now: number = Date.now()): string {
	return formatElapsed(now - new Date(startedAt).getTime());
}
