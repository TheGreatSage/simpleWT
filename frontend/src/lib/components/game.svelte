<script lang="ts">
	import { ConnectAllHandlers } from '$lib/handlers/handlers';
	// import { OpCodes } from "$lib/handlers/opcodes";
	import { Client } from '$lib/stores/client.svelte';
	import Map from '$lib/components/map.svelte';
	import { wtStore } from '$lib/stores/wt.svelte';
	import { OpCodes } from '$lib/handlers/opcodes';
	import { GameClientChat } from '$lib/cpnp/game';

	ConnectAllHandlers();

	let msgs = $derived(Client.messages);

	let chat_v: string = $state('');

	function chat() {
		wtStore.SendStreamMessage(OpCodes.CChat, GameClientChat, {
			text: chat_v
		});
		chat_v = '';
	}
</script>

<div class="divider">
	<div class="chat">
		<div class="grow"></div>
		{#each msgs as msg, idx (idx)}
			<div class="message">
				{msg}
			</div>
		{/each}
		<div class="chat-input">
			<input placeholder="chat..." bind:value={chat_v} />
			<button title="chat" onclick={chat}>Submit</button>
		</div>
	</div>
	<div class="game">
		<div class="grow"></div>
		<Map />
		<div class="grow"></div>
	</div>
</div>

<div class="garbage">
	<p>
		Garbage sending: {Client.garbage.amount} hashes {Client.garbage.per} per second --- Sent: {Client
			.garbage.sent}
	</p>
</div>

<style>
	.divider {
		display: flex;
		flex-direction: row;
		height: 90vh;
		width: 100vw;
	}
	.chat {
		display: flex;
		margin-right: auto;
		flex-direction: column;
		width: 50%;
	}
	.chat-input {
		white-space: nowrap;
	}
	.chat-input > input {
		width: 75%;
	}
	.grow {
		display: block;
		flex-grow: 1;
	}

	.game {
		/* flex-grow: 1; */
		/* align-items: center; */
		/* justify-content: center; */
		display: flex;
		flex-direction: column;
	}
	.garbage {
		text-align: center;
		font-size: smaller;
	}
</style>
