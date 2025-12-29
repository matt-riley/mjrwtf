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
- This site is deployed to the custom domain **https://docs.mjr.wtf**.
- `public/CNAME` is included so GitHub Pages applies the custom domain.
- Docs are **"latest" only** and are deployed from `main` (no versioned docs yet).
