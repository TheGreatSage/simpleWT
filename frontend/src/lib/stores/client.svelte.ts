import type { Player } from "$lib/cpnp/game";

// This should probably be a class.
let garbageActive: boolean = $state(false);
let garbageAmount: number = $state(0);
let garbagePer: number = $state(0);
let garbageBase: string = $state("");

let players: {[id: string]: {name: string, x: number, y: number} }= $state({});

// This is dumb
let messages: string[] = $state([]);

export function Client() {
    return {
        get messages() {
            return messages;
        },
        addMessage(msg: string) {
            messages.push(msg);
        },
        setGarbage(base: string, amount: number, per_second: number) {
            garbageBase = base;
            garbageAmount = amount;
            garbagePer = per_second;
            if (garbageAmount <= 0 || garbagePer <= 0) {
                garbageActive = false;
            } else {
                garbageActive = true;
            }
        },
        playerConnect(player: Player, connect: boolean) {
            if (connect) {
                players[player.id] = {
                    name: player.name,
                    x: player.x,
                    y: player.y,
                }
                this.addMessage(`Player ${player.name} connected.`);
            } else {
                this.addMessage(`Player ${player.name} disconnected.`);
                delete players[player.id];
            }
        },
        set players(list: {[id: string]: {name: string, x: number, y: number} }) {
            players = list;
        }
    }
}