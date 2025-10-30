import { Heartbeat } from "$lib/cpnp/control";
import { GameBroadcastChat, GameBroadcastConnect, GameBroadcastPlayerMove, GameServerGarbage, GameServerPlayers } from "$lib/cpnp/game";
import { Client } from "$lib/stores/client.svelte";
import { opHandlers } from "$lib/stores/handlers.svelte";
import { SendStreamMessage } from "$lib/stores/wt.svelte";
import { OpCodes } from "./opcodes";


// This is probably dumb too.
export function ConnectAllHandlers() {
    const handler = opHandlers();

    // Utility
    handler.addHandler(OpCodes.Heartbeat, Heartbeat, HandlePing);
    // Broadcasts
    handler.addHandler(OpCodes.BConnect, GameBroadcastConnect, HandleBConnect);
    handler.addHandler(OpCodes.BPlayerMoved, GameBroadcastPlayerMove, HandleBPlayerMoved);
    handler.addHandler(OpCodes.BChat, GameBroadcastChat, HandleBChat);
    // Server
    handler.addHandler(OpCodes.SGarbage, GameServerGarbage, HandleGarbageRequest);
    handler.addHandler(OpCodes.SPlayers, GameServerPlayers, HandleServerList)
}

function HandlePing(msg: Heartbeat) {
    if (!msg) {
        return;
    }
    SendStreamMessage(OpCodes.Heartbeat, Heartbeat, null);
}

function HandleBConnect(msg: GameBroadcastConnect) {
    if (!msg) {
        return;
    }
    if (!msg._hasPlayer()) {
        return;
    }
    
    Client().playerConnect(msg.player, msg.connected);
}

function HandleBPlayerMoved(msg: GameBroadcastPlayerMove) {
    if (!msg) {
        return;
    }
    if (!msg._hasWho()) {
        return;
    }
    // Setup canvas
}

function HandleBChat(msg: GameBroadcastChat) {
    if (!msg) {
        return;
    }
    // What is sanitization?
    Client().addMessage(`${msg.name}: ${msg.text}`);
}

function HandleGarbageRequest(msg: GameServerGarbage) {
    if (!msg) {
        return;
    }
    if (msg.base === '' || msg.base === undefined) {
        console.error('garbage reqeust has no base!');
        return;
    }
    Client().setGarbage(msg.base, msg.amount, msg.per);
}

function HandleServerList(msg: GameServerPlayers) {
    if (!msg) {
        return;
    }
    let players: {[id: string]: {name: string, x: number, y: number} } = {}
    for (let i = 0; i < msg.players.length; i++) {
        const pl = msg.players[i];
        players[pl.id] = {name: pl.name, x: pl.x, y: pl.y};
    }
    Client().players = players;
    Client().addMessage(`There are ${msg.players.length-1} connected.`);
}