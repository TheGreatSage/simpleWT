<script lang="ts">
	import Game from '$lib/components/game.svelte';
	import { wtStore } from '$lib/stores/wt.svelte';

	let spinner = $state(false);

	const transport = $derived(wtStore.transport);

	async function handleLogin(
		event: SubmitEvent & { currentTarget: EventTarget & HTMLFormElement }
	) {
		event.preventDefault();
		spinner = true;
		const data = new FormData(event.currentTarget, event.submitter);
		console.log(data);

		// probably the wrong way to do this.
		const wtip_f = data.get('wt-ip');
		const wtport_f = data.get('wt-port');
		if (!wtip_f || !wtport_f) {
			spinner = false;
			return;
		}
		const wtip = wtip_f.toString();
		const wtport = wtport_f.toString();

		const code = await fetch(
			`http://${data.get('login-ip')}:${data.get('login-port')}/login?name=${data.get('name')}`,
			{
				method: 'GET'
			}
		)
			.then((r) => {
				if (r.status !== 200) {
					return '';
				}
				console.log('login response', r);
				return r.text();
				// return {code: await r.text()};
			})
			.catch((e) => {
				console.log('login error', e);
				return;
			});

		if (!code || code === '' || code === undefined || code === 'undefined') {
			spinner = false;
			return;
		}

		console.log('code', code);

		const con = await wtStore.connect(code, wtip, wtport);
		console.log('connection', con);

		spinner = false;
	}
</script>

{#if spinner}
	<div class="center">
		<div class="loading">Loading</div>
		<span class="loader"></span>
	</div>
{:else if transport}
	<Game />
{:else}
	<form method="GET" onsubmit={handleLogin}>
		<!-- I am a web developer. How can you tell? -->
		Username:
		<input name="name" placeholder="Sage" required />
		<br />
		<br />
		Login IP:
		<input name="login-ip" value="127.0.0.1" required />
		Port:
		<input name="login-port" type="number" value="8770" required />
		<br />
		<br />
		Webtransport IP:
		<input name="wt-ip" value="127.0.0.1" required />
		Webtransport Port:
		<input name="wt-port" type="number" value="8771" required />
		<br />
		<br />
		<button>Login</button>
	</form>
{/if}
