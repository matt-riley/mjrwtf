# mjr.wtf docs site

This directory contains the Astro + Starlight documentation site for **mjr.wtf**.

## Local development

```bash
cd docs-site
npm install
npm run dev
```

## Deployment

- GitHub Pages deployment is automated via `.github/workflows/docs-pages.yml`.
- The site is built to work under the project pages base path: `/<repo>/` (currently `/mjrwtf/`).
- Docs are **"latest" only** and are deployed from `main` (no versioned docs yet).
