# Browser E2E tests (Playwright)

This folder contains **real browser** E2E tests for the HTML UI.

## Run

Prereqs:
- Go
- Node.js
- `sqlite3` available on your PATH (for DB assertions)

```bash
cd e2e
npm install
npx playwright install --with-deps chromium
npm test
```

The tests start the Go server on an ephemeral port with an isolated temp SQLite database, then clean up automatically.
