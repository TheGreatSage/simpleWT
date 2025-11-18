import type { OpCodes } from '$lib/handlers/opcodes';
import * as cpnp from 'capnp-es';

class HandlerStore {
	#handlers: Map<number, (payload: Uint8Array) => void> = $state(
		new Map<number, (payload: Uint8Array) => void>()
	);

	constructor() {
		// Needed?
	}

	addHandler = <T extends cpnp.Struct>(
		opcode: OpCodes,
		struct: Parameters<cpnp.Message['initRoot']>[0] & { prototype: T },
		handler: (msg: T) => void
	) => {
		this.#handlers.set(opcode, (buf: Uint8Array) => {
			try {
				const reader = new cpnp.Message(buf, false);
				const root = reader.getRoot(struct);
				handler(root as T);
			} catch (e) {
				console.error('Decode opcode error.', opcode, e);
			}
		});
	};

	handle = (opcode: number, payload: Uint8Array) => {
		const fun = this.#handlers.get(opcode);
		if (fun) {
			fun(payload);
		}
	};
}

export const opHandlers = new HandlerStore();
