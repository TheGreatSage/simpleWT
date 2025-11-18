import * as cpnp from 'capnp-es';
/* eslint-disable @typescript-eslint/no-explicit-any */

// From https://github.com/knervous/eqrequiem
export function setStructFields<T extends cpnp.Struct>(
	struct: T,
	data: Partial<Record<keyof T, any>>
) {
	for (const [rawKey, value] of Object.entries(data)) {
		if (value === undefined) {
			continue;
		}
		const key = rawKey as keyof T;

		// 1) Detect a JS array → list case
		if (Array.isArray(value)) {
			// build the "initArgs" method name
			const initName = `_init${String(key)[0].toUpperCase()}${String(key).slice(1)}`;
			const initFn = (struct as any)[initName] as ((n: number) => any) | undefined;
			if (typeof initFn === 'function') {
				const listBuilder = initFn.call(struct, value.length);
				for (let i = 0; i < value.length; i++) {
					listBuilder.set(i, value[i]);
				}
				continue;
			}
			// else fall‐through: maybe you have a byte‐list or something else
		}

		// 2) Fallback: simple scalar/struct assignment
		(struct as any)[key] = value;
	}
}
