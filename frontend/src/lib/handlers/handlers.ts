import {
	Heartbeat,
	GameBroadcastChat,
	GameBroadcastConnect,
	GameBroadcastPlayerMove,
	GameServerGarbage,
	GameServerGarbageAck,
	GameServerPlayers,
	Player
} from '$lib/beop/bops.gen';
import { Client } from '$lib/stores/client.svelte';
import { opHandlers } from '$lib/stores/handlers.svelte';
import { wtStore } from '$lib/stores/wt.svelte';
import { OpCodes } from './opcodes';

// This is probably dumb too.
export function ConnectAllHandlers() {
	// Utility
	opHandlers.addHandler(OpCodes.Heartbeat, Heartbeat, HandlePing);
	// Broadcasts
	opHandlers.addHandler(OpCodes.BConnect, GameBroadcastConnect, HandleBConnect);
	opHandlers.addHandler(OpCodes.BPlayerMoved, GameBroadcastPlayerMove, HandleBPlayerMoved);
	opHandlers.addHandler(OpCodes.BChat, GameBroadcastChat, HandleBChat);
	// Server
	opHandlers.addHandler(OpCodes.SGarbage, GameServerGarbage, HandleGarbageRequest);
	opHandlers.addHandler(OpCodes.SGarbageAck, GameServerGarbageAck, HandleServerGarbageAck);
	opHandlers.addHandler(OpCodes.SPlayers, GameServerPlayers, HandleServerList);
}

function HandlePing(msg: Heartbeat) {
	if (!msg) {
		return;
	}
	wtStore.SendStreamMsg(
		OpCodes.Heartbeat,
		Heartbeat({
			unix: BigInt(Date.now())
		})
	);
}

function HandleBConnect(msg: GameBroadcastConnect) {
	if (!msg) {
		return;
	}
	// Hope that the first connection sets the player.
	if (!Client.user) {
		Client.user = msg.player;
	} else {
		Client.connect(msg.player, msg.connected);
	}
}

function HandleBPlayerMoved(msg: GameBroadcastPlayerMove) {
	if (!msg) {
		return;
	}
	Client.move(msg.who);
}

function HandleBChat(msg: GameBroadcastChat) {
	if (!msg) {
		return;
	}
	// What is sanitization?
	Client.messages.push(`${msg.name}: ${msg.text}`);
}

function HandleGarbageRequest(msg: GameServerGarbage) {
	if (!msg) {
		return;
	}
	if (msg.base == null || msg.base === undefined || msg.base.length != 20) {
		console.error('garbage reqeust has no base!', msg.base);
		return;
	}
	Client.setGarbage(msg.amount, msg.per, msg.base);
}

function HandleServerGarbageAck(msg: GameServerGarbageAck) {
	if (!msg) {
		return;
	}
	// Probably could do more here.
	Client.garbageAck();
}

function HandleServerList(msg: GameServerPlayers) {
	if (!msg) {
		return;
	}
	const players: Player[] = [];
	for (let i = 0; i < msg.players.length; i++) {
		players.push(msg.players[i]);
		// console.log('player', i, msg.players[i].id);
	}
	Client.players = players;
	Client.messages.push(`There are ${msg.players.length - 1} others connected.`);
}
