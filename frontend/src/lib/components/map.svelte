<script lang="ts">
	import { GameClientMoved, Player } from '$lib/beop/bops.gen';
	import { OpCodes } from '$lib/handlers/opcodes';
	import { Client } from '$lib/stores/client.svelte';
	import { wtStore } from '$lib/stores/wt.svelte';
	import type { Attachment } from 'svelte/attachments';

	let canvas: HTMLCanvasElement;
	function drawGrid(players: Player[]): Attachment {
		return () => {
			console.log('drawing canvas');
			const ctx = canvas.getContext('2d');
			if (!ctx) {
				console.log('no ctx');
				return;
			}

			ctx.clearRect(0, 0, canvas.width, canvas.height);

			const width = Math.floor(canvas.width / 100);
			const height = Math.floor(canvas.height / 100);

			const pattern = grid(width, height);
			// canvas.width = devicePixelRatio * canvas.clientWidth;

			if (pattern) {
				const fill = ctx.createPattern(pattern, 'repeat');
				if (fill) {
					ctx.fillStyle = fill;
					ctx.fillRect(0, 0, canvas.width, canvas.height);
				}
			}

			const user = Client.user;

			if (!user) {
				console.log('no user');
				return;
			}
			const len = players.length;
			for (let i = 0; i < len; i++) {
				const pl = players[i];
				ctx.fillStyle = 'red';
				if (user.ID === pl.ID) {
					ctx.fillStyle = 'blue';
				}
				ctx.beginPath();
				ctx.ellipse(
					pl.X * width + width / 2,
					pl.Y * height + height / 2,
					width / 2,
					height / 2,
					0,
					0,
					2 * Math.PI
				);
				ctx.fill();
			}
		};
	}

	// Create a pattern canvas for the grid
	function grid(x: number, y: number): HTMLCanvasElement | null {
		const patternCanvas = document.createElement('canvas');
		const patternCtx = patternCanvas.getContext('2d');

		if (!patternCtx) {
			return null;
		}

		patternCanvas.width = x;
		patternCanvas.height = y;

		patternCtx.strokeStyle = 'black';
		patternCtx.lineWidth = 1;

		patternCtx.moveTo(x, 0);
		patternCtx.lineTo(x, y);
		patternCtx.moveTo(0, y);
		patternCtx.lineTo(x, y);

		patternCtx.stroke();

		return patternCanvas;
	}

	function move(x: number, y: number) {
		wtStore.SendStreamMsg(
			OpCodes.CMoved,
			GameClientMoved({
				X: x,
				Y: y
			})
		);
	}
</script>

<canvas class="map" width="800" height="600" bind:this={canvas} {@attach drawGrid(Client.players)}
></canvas>

<div class="movement">
	<button
		class="mv left"
		onclick={() => {
			move(-1, 0);
		}}>&lt;</button
	>
	<button
		class="mv up"
		onclick={() => {
			move(0, -1);
		}}>&lt;</button
	>
	<button
		class="mv down"
		onclick={() => {
			move(0, 1);
		}}>&gt;</button
	>
	<button
		class="mv right"
		onclick={() => {
			move(1, 0);
		}}>&gt;</button
	>
</div>

<style>
	.movement {
		display: flex;
		flex-direction: row;
		justify-content: center;
		margin-right: 4em;
		margin-top: 0.5em;
		gap: 0.5em;
	}
	.mv {
		height: 2em;
		width: 2em;
	}
	.up {
		writing-mode: vertical-rl;
	}
	.down {
		writing-mode: vertical-rl;
	}
	.map {
		/* flex-grow: 1;
        /* display: block; */
		/* position: relative; */
		/* object-fit: none; */
		width: 600px;
		/* margin: auto; */
		margin-right: 4em;
		border: lightgray 1px solid;
		border-radius: 3px;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
	}
</style>
