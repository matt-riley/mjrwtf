// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://matt-riley.github.io',
	base: '/mjrwtf',
	integrations: [
		starlight({
			title: 'mjr.wtf',
			description: 'Documentation for the mjr.wtf URL shortener.',
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/matt-riley/mjrwtf' }],
			sidebar: [
				{ label: 'Getting Started', autogenerate: { directory: 'getting-started' } },
				{ label: 'API', autogenerate: { directory: 'api' } },
				{ label: 'Operations', autogenerate: { directory: 'operations' } },
				{ label: 'Contributing', autogenerate: { directory: 'contributing' } },
				{ label: 'Security', slug: 'security' },
			],
		}),
	],
});
