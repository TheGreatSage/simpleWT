import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/hash': {
				target: 'http://localhost:8775/hash',
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/hash/, '')
			}
		}
	}
});
