import { b64decode } from '$lib/utils/base64';
import { opHandlers } from './handlers.svelte';
import { Client } from './client.svelte';
import { Uint8ArrayConcat } from '$lib/utils/uint8array';
import type { BebopRecord } from 'bebop';

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

// let webtransport = $state<WebTransport | null>(null);
// let writer = $state<WritableStreamDefaultWriter<Uint8Array> | null>(null);

class WebTransportStore {
	transport: WebTransport | null = $state(null);
	writer: WritableStreamDefaultWriter | null = $state(null);

	constructor() {
		// Do something here?
	}

	connect = async (code: string, url: string, port: number | string): Promise<boolean> => {
		if (!code || code === '') {
			return false;
		}

		if (this.transport) {
			console.log('created');
			const closed = await this.transport.closed.catch(() => null);
			if (!closed) {
				// close
			}
		}

		try {
			// Don't do this
			const hash = await fetch('hash').then((r: Response) => r.text());
			console.log('Got Hash', hash);
			this.transport = new WebTransport(`https://${url}:${port}/wt?code=${code}`, {
				serverCertificateHashes: [{ algorithm: 'sha-256', value: b64decode(hash) }]
			});
		} catch (e) {
			this.reset();
			console.warn('failed', e);
			return false;
		}

		console.log('Waiting for webtransport ready.');
		try {
			await this.transport.ready;
		} catch (e) {
			this.reset();
			console.warn('failed webtransport ready', e);
			return false;
		}

		console.log('Webtransport Ready');

		// Probably a better place for this.
		Client.reset();

		// Streams
		this.bidirectionalStream();

		return true;
	};

	bidirectionalStream = async () => {
		if (!this.transport) {
			console.log("no streams if transport is doesn't exist");
			return;
		}

		// Ideally this would better handle multiple streams.
		// It might, but it should track stuff better.
		try {
			const stream = this.transport.incomingBidirectionalStreams.getReader();
			while (true) {
				stream.closed.then(() => {
					console.log('stream closed');
				});
				const { done, value } = await stream.read();
				if (done) {
					break;
				}
				console.log('Starting stream');
				this.writer = value.writable.getWriter();

				// Read Stream
				this.#readStream(value.readable);
			}
		} catch (e) {
			if (e instanceof WebTransportError) {
				if (e.message === 'remote WebTransport close') {
					console.log('Steam closed');
				}
			} else {
				console.error('Stream error', e);
			}
			return;
		}
	};

	SendStreamMsg = async (opcode: number, msg: BebopRecord) => {
		if (!this.writer) {
			this.reset();
			throw new Error('Stream not open!');
		}

		const payload = msg.encode();

		// [opcode:u16_le][length:u32_le]
		const header = new ArrayBuffer(6);
		// can you use re-use DataView?
		new DataView(header).setUint16(0, opcode, true);
		new DataView(header).setUint32(2, payload.length, true);

		// [header][payload]
		const frame = Uint8ArrayConcat(new Uint8Array(header), payload);

		// console.log("frame", frame);
		await this.writer.write(frame);
	};

	#readStream = async (stream: ReadableStream<Uint8Array>) => {
		const rdr = stream.getReader();
		try {
			let buffer: Uint8Array = new Uint8Array();
			while (this.transport) {
				const { value, done } = await rdr.read();
				if (done) {
					break;
				}
				if (!value) {
					continue;
				}
				buffer = Uint8ArrayConcat(buffer, value);
				while (buffer.length >= 6) {
					const op = new DataView(buffer.buffer).getUint16(0, true);
					const len = new DataView(buffer.buffer).getUint32(2, true);
					if (buffer.length < 6 + len) {
						break;
					}
					const payload = buffer.slice(6, 6 + len);

					opHandlers.handle(op, payload);

					buffer = buffer.slice(6 + len);
				}
			}
		} catch (e) {
			if (e instanceof WebTransportError && e.message === 'WebTransportStream Reset') {
				// Ignore
			} else {
				console.log('reader error: ', e);
			}
			// Does this need to be reset here?
			this.reset();
		} finally {
			rdr.releaseLock();
		}
	};

	reset = () => {
		this.transport = null;
		this.writer = null;
	};
}

export const wtStore = new WebTransportStore();
