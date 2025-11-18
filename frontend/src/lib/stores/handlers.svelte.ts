import type { OpCodes } from '$lib/handlers/opcodes';
import type { BebopRecord } from 'bebop';

interface Decoder<T> {
	decode(buffer: Uint8Array): T & BebopRecord;
}

class HandlerStore {
	#handlers: Map<number, (payload: Uint8Array) => void> = $state(
		new Map<number, (payload: Uint8Array) => void>()
	);

	constructor() {
		// Needed?
	}

	addHandler = <T>(opcode: OpCodes, struct: Decoder<T>, handler: (msg: T) => void) => {
		this.#handlers.set(opcode, (buf: Uint8Array) => {
			try {
				handler(struct.decode(buf));
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
