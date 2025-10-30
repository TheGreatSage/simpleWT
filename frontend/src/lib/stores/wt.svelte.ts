import { b64decode } from '$lib/utils/base64';
import { opHandlers } from './handlers.svelte';
import * as cpnp from 'capnp-es';
import { setStructFields } from '$lib/utils/capnp';

// Dummy interface for type checking WebTransport
interface WebTransport {
	readonly closed: Promise<{ closeCode?: number; reason?: string }>;
	// congestionControl
	readonly datagrams: {
		readonly writable: WritableStream<Uint8Array>;
		readonly readable: ReadableStream<Uint8Array>;
	};
	readonly incomingBidirectionalStreams: ReadableStream<{
		readable: ReadableStream<Uint8Array>;
		writable: WritableStream<Uint8Array>;
	}>;
	// incomingUnidirectionalStreams
	readonly ready: Promise<void>;
	// reliability
}


let webtransport = $state<WebTransport | null>(null);
let writer = $state<WritableStreamDefaultWriter<Uint8Array> | null>(null);

export function wtStore() {
	return {
		get transport() {
			return webtransport;
		}
	}
}

export async function Connect(code: string, url: string, port: number | string): Promise<boolean> {
	if (!code || code === '') {
		return false;
	}

	if (webtransport) {
		console.log('created');
		const closed = await webtransport.closed.catch(() => null);
		if (!closed) {
			// close
		}
	}

	try {
		// SDon't do this
		const hash = await fetch('hash').then((r: Response) => r.text());
		console.log('Got Hash', hash);
		webtransport = new WebTransport(`https://${url}:${port}/wt?code=${code}`, {
			serverCertificateHashes: [{ algorithm: 'sha-256', value: b64decode(hash) }]
		});
	} catch (e) {
		webtransport = null;
		console.warn('failed', e);
		return false;
	}

	console.log('Readying');
	try {
		await webtransport.ready;
	} catch (e) {
		webtransport = null;
		console.warn('failed', e);
		return false;
	}

	console.log('Ready');

	// Streams
	bidirectionalStream(webtransport);

	webtransport.closed
		.then(() => {
			console.log('WebTransport closed gracefully.');
		})
		.catch((e) => {
			console.log('WebTransport connection closed with error:', e);
		});


	return true;
}

function concatUint8(a: Uint8Array, b: Uint8Array): Uint8Array {
  const c = new Uint8Array(a.length + b.length);
  c.set(a, 0);
  c.set(b, a.length);
  return c;
}


async function controlReader(stream: ReadableStream<Uint8Array>) {
	const rdr = stream.getReader();
	try {
		let buffer: Uint8Array = new Uint8Array();
		const ops = opHandlers();
		while (true) {
			const { value, done } = await rdr.read();
			if (done) {
				break;
			}
			if (!value) {
				continue;
			}
			buffer = concatUint8(buffer, value);
			while (buffer.length >= 6) {
				const op = new DataView(buffer.buffer).getUint16(0, true);
				const len = new DataView(buffer.buffer).getUint32(2, true);
				if (buffer.length < 6 + len) {
					break;
				}
				const payload = buffer.slice(6, 6 + len);

				ops.handle(op, payload);

				buffer = buffer.slice(6 + len);
			}
		}
	} catch (e) {
		console.log('reader error: ', e);
	} finally {
		rdr.releaseLock();
	}
}

async function bidirectionalStream(transport: WebTransport) {
	const stream = transport.incomingBidirectionalStreams.getReader();
	while (true) {
		stream.closed.then(() => {
			console.log('stream closed');
		});
		const { done, value } = await stream.read();
		if (done) {
			break;
		}
		console.log('got stream');
		writer = value.writable.getWriter();

		

		// Readable
		controlReader(value.readable);
	}
}

export async function SendStreamMessage<T extends cpnp.Struct>(
	opcode: number,
	struct: Parameters<cpnp.Message['initRoot']>[0] & { prototype: T },
	data: Partial<Record<keyof T, any>> | null,
): Promise<void> {
	if (!writer) {
		throw new Error('Stream not open!');
	}
	const msg = new cpnp.Message();
	const root = msg.initRoot(struct);
	if (data) {
		setStructFields(root, data);
	}

	const payload = new Uint8Array(cpnp.Message.toArrayBuffer(msg));

	// [opcode:u16_le][length:u32_le]
	const header = new ArrayBuffer(6);
	// can you use re-use DataView?
	new DataView(header).setUint16(0, opcode, true);
	new DataView(header).setUint32(2, payload.length, true);

	// [header][payload]
	const frame = concatUint8(
		new Uint8Array(header),
		payload
	);

	console.log("frame", frame);
	await writer.write(frame);
}