# Browser E2E tests (Playwright)

This folder contains **real browser** E2E tests for the HTML UI.

## Run

Prereqs:
- Docker + Docker Compose
- Node.js
- `sqlite3` available on your PATH (for DB assertions)

```bash
cd e2e
npm install
npx playwright install --with-deps chromium
npm test
```

The tests start `docker compose` with an isolated temp `DATA_DIR` and an ephemeral `HOST_PORT`, then clean up automatically.
