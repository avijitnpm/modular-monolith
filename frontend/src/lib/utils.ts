type ClassValue = string | number | boolean | null | undefined | ClassValue[] | Record<string, boolean>;

export function cn(...inputs: ClassValue[]) {
	return inputs.flatMap(classNames).filter(Boolean).join(" ");
}

function classNames(input: ClassValue): string[] {
	if (!input) {
		return [];
	}

	if (Array.isArray(input)) {
		return input.flatMap(classNames);
	}

	if (typeof input === "object") {
		return Object.entries(input)
			.filter(([, enabled]) => enabled)
			.map(([name]) => name);
	}

	return [String(input)];
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChild<T> = T extends { child?: any } ? Omit<T, "child"> : T;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type WithoutChildren<T> = T extends { children?: any } ? Omit<T, "children"> : T;
export type WithoutChildrenOrChild<T> = WithoutChildren<WithoutChild<T>>;
export type WithElementRef<T, U extends HTMLElement = HTMLElement> = T & { ref?: U | null };
