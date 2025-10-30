// Different Methods for base64 to string from a random stack overflow post.
// Have them all in here just because.

// This should be the fastest use of atob.
export function b64ToArrayBuffer(base64: string): ArrayBuffer {
	const bstring = atob(base64);
	const bytes = new Uint8Array(bstring.length);
	for (let i = 0; i < bstring.length; i++) {
		bytes[i] = bstring.charCodeAt(i);
	}
	return bytes.buffer;
}

// This is slower than the for loop.
export function b64ArrayBuffer(base64: string): ArrayBuffer {
	return Uint8Array.from(base64, (c) => c.charCodeAt(0)).buffer;
}

// Should be the fastest way. Which is reimplementing atob.

// https://stackoverflow.com/questions/21797299/how-can-i-convert-a-base64-string-to-arraybuffer#comment134219283_41106346
// https://github.com/niklasvh/base64-arraybuffer/blob/master/src/index.ts#L31
const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';

const lookup = new Uint8Array(256);
for (let i = 0; i < chars.length; i++) {
	lookup[chars.charCodeAt(i)] = i;
}

export function b64encode(array: ArrayBuffer): string {
	const bytes = new Uint8Array(array);
	const len: number = bytes.length;

	let base64: string = '';
	for (let i = 0; i < len; i += 3) {
		base64 += chars[bytes[i] >> 2];
		base64 += chars[((bytes[i] & 3) << 4) | (bytes[i + 1] >> 4)];
		base64 += chars[((bytes[i + 1] & 15) << 2) | (bytes[i + 2] >> 6)];
		base64 += chars[bytes[i + 2] & 63];
	}

	if (len === 2) {
		base64 = base64.substring(0, base64.length - 1) + '=';
	} else if (len % 3 === 1) {
		base64 = base64.substring(0, base64.length - 2) + '==';
	}

	return base64;
}

export function b64decode(base64: string): ArrayBuffer {
	const len = base64.length;
	let buff = len * 0.75;
	if (base64[len - 1] === '=') {
		buff--;
		if (base64[len - 2] === '=') {
			buff--;
		}
	}

	const array = new ArrayBuffer(buff);
	const bytes = new Uint8Array(array);

	let e1: number;
	let e2: number;
	let e3: number;
	let e4: number;
	let p: number = 0;

	for (let i = 0; i < len; i += 4) {
		e1 = lookup[base64.charCodeAt(i)];
		e2 = lookup[base64.charCodeAt(i + 1)];
		e3 = lookup[base64.charCodeAt(i + 2)];
		e4 = lookup[base64.charCodeAt(i + 3)];

		bytes[p++] = (e1 << 2) | (e2 >> 4);
		bytes[p++] = ((e2 & 15) << 4) | (e3 >> 2);
		bytes[p++] = ((e3 & 3) << 6) | (e4 & 63);
	}

	return array;
}
