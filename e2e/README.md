# Browser E2E tests (Playwright)

This folder contains **real browser** E2E tests for the HTML UI.

The tests start the Go server on an ephemeral port with an isolated, temp **file-backed** SQLite database, then clean up automatically.

## Run

Prereqs:
- Go
- Node.js
- `sqlite3` available on your PATH (used for DB assertions)

From repo root, make sure generated code is up-to-date:

```bash
make generate
```

Then run Playwright:

```bash
cd e2e
npm ci
npx playwright install chromium
npm test
```

Linux CI note: `npx playwright install --with-deps chromium`.
