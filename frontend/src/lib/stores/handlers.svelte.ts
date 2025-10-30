
import type { OpCodes } from '$lib/handlers/opcodes';
import * as cpnp from 'capnp-es';

let handlers: {[opcode: number]: (payload: Uint8Array) => void; } = $state({}); 

export function opHandlers() {
    return {
        get handlers() {
            return handlers;
        },
        addHandler<T extends cpnp.Struct>(
            opcode: OpCodes,
            struct: Parameters<cpnp.Message['initRoot']>[0] & { prototype: T },
            handler: (msg: T) => void,
        ) {
            handlers[opcode] = (buf: Uint8Array) => {
                try {
                    const reader = new cpnp.Message(buf, false);
                    const root = reader.getRoot(struct);
                    handler(root as T);
                } catch (e) {
                    console.error("Decode opcode error.", opcode, e);
                }
            }
        },
        handle(opcode: number, payload: Uint8Array) {
            handlers[opcode]?.(payload);
        },

    }
} 