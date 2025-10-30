<script lang="ts">
	import { ConnectAllHandlers } from "$lib/handlers/handlers";
	import { OpCodes } from "$lib/handlers/opcodes";
	import { Client } from "$lib/stores/client.svelte";
	import { opHandlers } from "$lib/stores/handlers.svelte";

    
    ConnectAllHandlers();
    const handlers = opHandlers();
    // handlers.addHandler(OpCodes.Heartbeat, )

    let msgs = $derived(Client().messages);
</script>

<div class="divider">
    <div class="chat">
        <div class="grow">
        </div>
        {#each msgs as msg}
            <div class="message">
                {msg}
            </div>
        {/each}
        <div class="chat-input">
            <input placeholder="chat..."/>
            <button title="chat">Submit</button> 
        </div>
        
    </div>
    <div class="game">
        <!-- Canvas here -->
    </div>
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
        flex-direction: column;
        width: 50%;
    }
    .grow {
        flex-grow: 1;
    }
</style>