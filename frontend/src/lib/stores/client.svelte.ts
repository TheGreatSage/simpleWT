import { GameClientGarbage, GarbageData, Player } from '$lib/beop/bops.gen';
import { OpCodes } from '$lib/handlers/opcodes';
import { Uint8ArrayConcat } from '$lib/utils/uint8array';
import { wtStore } from './wt.svelte';

class GarbageStore {
	amount: number = $state(0);
	per: number = $state(0);
	base: Uint8Array = $state(new Uint8Array());
	sent: number = $state(0);

	wait: boolean = $state(true);

	#timeoutID: number | undefined = undefined;

	constructor() {
		// something
	}

	async #msg(num: number): Promise<GarbageData> {
		// This can't be the way to hash a function in js can it?
		const numbarr = new TextEncoder().encode(`${num}`);
		const msg = Uint8ArrayConcat(this.base, numbarr);
		const hashBuf = await crypto.subtle.digest('SHA-1', msg);
		// const hashArr = Array.from(new Uint8Array(hashBuf));
		// const hash = hashArr.map(b => b.toString(16).padStart(2, '0')).join('');

		// This right?
		return { data: new Uint8Array(hashBuf) };
	}

	#run = () => {
		// Miss a whole tick, oh well.
		// This should also probably count misses.
		if (this.wait) {
			return;
		}
		const hashes: Promise<GarbageData>[] = [];
		for (let i = 0; i < this.amount; i++) {
			hashes.push(this.#msg(i));
		}

		Promise.all(hashes)
			.then((garbage) => {
				try {
					// Oof this is rough.
					// Only way I could figure this out though.
					const msg = GameClientGarbage({ hashes: garbage });

					// console.log(gcg.hash.at(1).data.toString());
					wtStore.SendStreamMsg(OpCodes.CGarbage, msg).then(() => {
						this.sent++;
					});
					this.wait = true;
				} catch (e) {
					console.log('failed sending garbage', e);
					window.clearInterval(this.#timeoutID);
					this.reset(0, 0);
				}
			})
			.catch(() => {
				this.reset(0, 0);
			});
	};

	reset = (amount: number, per: number, base: Uint8Array | null = null) => {
		window.clearInterval(this.#timeoutID);
		this.amount = amount;
		this.per = per;
		if (base) {
			this.base = base;
		} else {
			this.base = new Uint8Array();
		}

		this.wait = true;
		if (amount <= 0 || per <= 0) {
			// Nothing
		} else {
			this.#timeoutID = window.setInterval(this.#run, 1000 / this.per);
		}
	};
}

class ClientStore {
	// This is a dumb way to do messages.
	messages: string[] = $state([]);
	user: Player | null = $state(null);
	// There is probably a bettter way to do this.
	#playerMap: Map<string, number> = new Map<string, number>();
	#players: Player[] = $state([]);

	garbage: GarbageStore = new GarbageStore();

	constructor() {
		// Something here?
	}

	get players(): Player[] {
		return this.#players;
	}

	set players(list: Player[]) {
		this.#players = list;
		this.#updatePlayerMap();
	}

	setGarbage = (amount: number, per: number, base: Uint8Array) => {
		this.garbage.reset(amount, per, base);
		this.garbage.wait = false;
	};

	garbageAck = () => {
		this.garbage.wait = false;
	};

	connect = (player: Player, connect: boolean) => {
		if (connect) {
			this.#playerMap.set(player.ID, this.#players.length);
			this.#players.push(player);
			this.messages.push(`Player ${player.name} connected.`);
		} else {
			const idx = this.#playerMap.get(player.ID);
			this.messages.push(`Player ${player.name} disconnected.`);
			if (idx) {
				delete this.#players[idx];
				this.#playerMap.delete(player.ID);
			}
		}
	};

	move = (player: Player) => {
		const idx = this.#playerMap.get(player.ID);
		if (idx === undefined) {
			console.log('no one to move');
			return;
		}
		this.#players[idx] = player;
	};

	reset = () => {
		this.messages = [];
		this.user = null;

		this.#playerMap.clear();
		this.#players = [];
	};

	// Probably a smarter way to do this.
	#updatePlayerMap = () => {
		this.#playerMap.clear();
		for (let i = 0; i < this.#players.length; i++) {
			this.#playerMap.set(this.#players[i].ID, i);
		}
	};
}

export const Client = new ClientStore();
