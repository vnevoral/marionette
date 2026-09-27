import type { RunEvent, StatusEvent } from "@/api";

// Minimal EventSource stand-in for unit tests: records instances, lets a test
// emit named events and observe close().
export class FakeEventSource {
	static instances: FakeEventSource[] = [];
	readonly url: string;
	closed = false;
	private listeners = new Map<string, Array<(event: Event) => void>>();

	constructor(url: string) {
		this.url = url;
		FakeEventSource.instances.push(this);
	}

	static reset() {
		FakeEventSource.instances = [];
	}

	static last(): FakeEventSource {
		const source = FakeEventSource.instances.at(-1);
		if (!source) throw new Error("no EventSource was created");
		return source;
	}

	addEventListener(type: string, listener: (event: Event) => void) {
		const existing = this.listeners.get(type) ?? [];
		this.listeners.set(type, [...existing, listener]);
	}

	close() {
		this.closed = true;
	}

	emit(type: string, event: Event = new Event(type)) {
		for (const listener of this.listeners.get(type) ?? []) listener(event);
	}

	open() {
		this.emit("open");
	}

	fail() {
		this.emit("error");
	}

	status(payload: StatusEvent) {
		this.emit(
			"status.changed",
			new MessageEvent("status.changed", { data: JSON.stringify(payload) }),
		);
	}

	run(payload: RunEvent) {
		this.emit("run.recorded", new MessageEvent("run.recorded", { data: JSON.stringify(payload) }));
	}
}
