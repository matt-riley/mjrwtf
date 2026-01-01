const { test, expect } = require('@playwright/test');
const childProcess = require('node:child_process');
const fs = require('node:fs');
const net = require('node:net');
const os = require('node:os');
const path = require('node:path');

const repoRoot = path.resolve(__dirname, '../..');

const POLL_INTERVAL_MS = 250;
const HEALTH_CHECK_TIMEOUT_MS = 120_000;
// Max time to wait for SQLite writes to become visible across connections; 10s is a
// generous upper bound chosen to keep e2e tests stable even on slow CI runners.
const DB_WRITE_PROPAGATION_TIMEOUT_MS = 10_000;

const SHORT_CODE_RE = /^[A-Za-z0-9_-]+$/;

function assertSafeShortCode(shortCode) {
  if (!SHORT_CODE_RE.test(shortCode)) {
    throw new Error('Invalid short code extracted from short URL');
  }
}

function execFileSyncQuiet(cmd, args, options) {
  return childProcess.execFileSync(cmd, args, {
    stdio: ['ignore', 'pipe', 'pipe'],
    ...options,
  });
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function getFreePort() {
  return await new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();
    server.on('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      server.close(() => resolve(address.port));
    });
  });
}

async function waitForHealthy(baseURL, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let logged = false;
  /** @type {unknown} */
  let lastErr;

  // Node 18+ has global fetch.
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseURL}/ready`);
      if (res.ok) return;
    } catch (err) {
      lastErr = err;
      if (!logged) {
        // Log once to aid debugging, but keep retrying until timeout.
        console.error('Readiness check request failed:', err?.name, err?.message);
        logged = true;
      }
    }

    await sleep(POLL_INTERVAL_MS);
  }

  throw new Error(`Timed out waiting for ${baseURL}/ready (lastErr=${lastErr})`);
}

const SQLITE_NO_ROW = '__NO_ROW__';

function sqliteGetOriginalURL(dbPath, shortCode) {
  assertSafeShortCode(shortCode);
  const out = execFileSyncQuiet(
    'sqlite3',
    [
      '-readonly',
      dbPath,
      '-cmd',
      '.parameter init',
      '-cmd',
      // Safe: shortCode is validated against SHORT_CODE_RE (alnum/_/- only).
      `.parameter set @short_code '${shortCode}'`,
      `SELECT CASE WHEN EXISTS(SELECT 1 FROM urls WHERE short_code=@short_code)
        THEN (SELECT original_url FROM urls WHERE short_code=@short_code LIMIT 1)
        ELSE '${SQLITE_NO_ROW}'
      END;`,
    ],
    { encoding: 'utf8' },
  );
  return out.trimEnd();
}

test.describe('UI E2E: /create persists to SQLite (docker compose)', () => {
  test.describe.configure({ mode: 'serial' });

  /** @type {{ projectName: string, containerName: string, dataDir: string, hostPort: number, authToken: string }} */
  const ctx = {
    projectName: `mjrwtf-e2e-${Date.now()}-${process.pid}`,
    containerName: '',
    dataDir: '',
    hostPort: 0,
    authToken: 'e2e-token',
  };

  test.beforeAll(async () => {
    ctx.hostPort = await getFreePort();
    ctx.containerName = `${ctx.projectName}-server`;

    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'mjrwtf-e2e-'));
    ctx.dataDir = path.join(tmpDir, 'data');
    fs.mkdirSync(ctx.dataDir, { recursive: true });

    const env = {
      ...process.env,
      COMPOSE_PROJECT_NAME: ctx.projectName,
      CONTAINER_NAME: ctx.containerName,
      HOST_PORT: String(ctx.hostPort),
      DATA_DIR: ctx.dataDir,
      AUTH_TOKENS: ctx.authToken,
    };

    execFileSyncQuiet('docker', ['compose', 'up', '-d', '--build'], { cwd: repoRoot, env });

    await waitForHealthy(`http://localhost:${ctx.hostPort}`, HEALTH_CHECK_TIMEOUT_MS);
  });

  test.afterAll(async () => {
    const env = {
      ...process.env,
      COMPOSE_PROJECT_NAME: ctx.projectName,
      CONTAINER_NAME: ctx.containerName,
      HOST_PORT: String(ctx.hostPort),
      DATA_DIR: ctx.dataDir,
    };

    try {
      execFileSyncQuiet('docker', ['compose', 'down', '--remove-orphans'], { cwd: repoRoot, env });
    } finally {
      if (ctx.dataDir) {
        // dataDir is <tmp>/data
        const tmpDir = path.dirname(ctx.dataDir);
        try {
          fs.rmSync(tmpDir, { recursive: true, force: true });
        } catch (err) {
          console.error(
            'Failed to remove temporary directory %s: %s',
            tmpDir,
            err?.message ?? String(err),
          );
        }
      }
    }
  });

  test('submitting /create writes urls row to file-backed DB', async ({ page }) => {
    const originalURL = `https://example.com/e2e/${Date.now()}`;
    const baseURL = `http://localhost:${ctx.hostPort}`;

    await page.goto(`${baseURL}/create`);
    await page.fill('#original_url', originalURL);
    await page.fill('#auth_token', ctx.authToken);

    await page.click('button[type="submit"]');

    const shortUrlInput = page.locator('#short-url-display');
    await expect(shortUrlInput).toBeVisible();

    const shortURL = (await shortUrlInput.inputValue()).trim();
    expect(shortURL).toMatch(/^https?:\/\//);

    // Expected format is {baseURL}/{shortCode}, but be tolerant of trailing slashes.
    const urlObj = new URL(shortURL);
    const trimmedPathname = urlObj.pathname.replace(/\/+$/, '');
    const pathSegments = trimmedPathname.split('/');
    const shortCode = pathSegments.pop() || '';
    assertSafeShortCode(shortCode);

    const dbPath = path.join(ctx.dataDir, 'database.db');

    const deadline = Date.now() + DB_WRITE_PROPAGATION_TIMEOUT_MS;
    let loggedQueryErr = false;
    while (Date.now() < deadline) {
      if (fs.existsSync(dbPath)) {
        try {
          const persisted = sqliteGetOriginalURL(dbPath, shortCode);
          if (persisted !== SQLITE_NO_ROW) {
            expect(persisted).toBe(originalURL);
            return;
          }
        } catch (err) {
          if (!loggedQueryErr) {
            console.error('DB query failed (will retry):', err?.message ?? String(err));
            loggedQueryErr = true;
          }
        }
      }

      await sleep(POLL_INTERVAL_MS);
    }

    throw new Error('Row not found in sqlite DB for created short URL');
  });

  test('dashboard requires a session and lists created URLs', async ({ page }) => {
    const baseURL = `http://localhost:${ctx.hostPort}`;

    // Unauthenticated users should be redirected to /login.
    await page.goto(`${baseURL}/dashboard`);
    await expect(page).toHaveURL(`${baseURL}/login`);
    await expect(page.getByRole('heading', { name: 'Dashboard Login' })).toBeVisible();

    // Login via the UI.
    await page.fill('#auth_token', ctx.authToken);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(`${baseURL}/dashboard`);
    await expect(page.getByRole('heading', { name: 'URL Dashboard' })).toBeVisible();

    // Create a URL via the UI.
    const originalURL = `https://example.com/e2e/dashboard/${Date.now()}`;

    await page.goto(`${baseURL}/create`);
    await page.fill('#original_url', originalURL);
    await page.fill('#auth_token', ctx.authToken);
    await page.click('button[type="submit"]');

    const shortUrlInput = page.locator('#short-url-display');
    await expect(shortUrlInput).toBeVisible();

    const shortURL = (await shortUrlInput.inputValue()).trim();
    const urlObj = new URL(shortURL);
    const trimmedPathname = urlObj.pathname.replace(/\/+$/, '');
    const pathSegments = trimmedPathname.split('/');
    const shortCode = pathSegments.pop() || '';
    assertSafeShortCode(shortCode);

    // Dashboard should show the created URL.
    await page.goto(`${baseURL}/dashboard`);
    const tableBody = page.locator('#urls-table-body');
    await expect(tableBody).toContainText(shortCode);
    await expect(tableBody).toContainText(originalURL);
  });
});
