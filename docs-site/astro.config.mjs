// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://docs.mjr.wtf',
	integrations: [
		starlight({
			title: 'mjr.wtf',
			description: 'Documentation for the mjr.wtf URL shortener.',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/matt-riley/mjrwtf' }],
			sidebar: [
				{ label: 'Getting Started', items: [{ autogenerate: { directory: 'getting-started' } }] },
				{ label: 'API', items: [{ autogenerate: { directory: 'api' } }] },
				{ label: 'Operations', items: [{ autogenerate: { directory: 'operations' } }] },
				{ label: 'Contributing', items: [{ autogenerate: { directory: 'contributing' } }] },
				{ label: 'Security', slug: 'security' },
			],
		}),
	],
});
