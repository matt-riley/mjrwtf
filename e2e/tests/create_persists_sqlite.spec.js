const { test, expect } = require('@playwright/test');
const childProcess = require('node:child_process');
const fs = require('node:fs');
const net = require('node:net');
const os = require('node:os');
const path = require('node:path');

const repoRoot = path.resolve(__dirname, '../..');

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

  // Node 18+ has global fetch.
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseURL}/health`);
      if (res.ok) return;
    } catch {
      // ignore
    }

    await sleep(250);
  }

  throw new Error(`Timed out waiting for ${baseURL}/health`);
}

function sqliteGetOriginalURL(dbPath, shortCode) {
  const out = execFileSyncQuiet(
    'sqlite3',
    ['-readonly', dbPath, `SELECT original_url FROM urls WHERE short_code='${shortCode}';`],
    { encoding: 'utf8' },
  );
  return out.trim();
}

test.describe('UI E2E: /create persists to SQLite (docker compose)', () => {
  test.setTimeout(10 * 60 * 1000);
  test.describe.configure({ mode: 'serial' });

  /** @type {{ projectName: string, containerName: string, dataDir: string, hostPort: number, authToken: string }} */
  const ctx = {
    projectName: `mjrwtf-e2e-${Date.now()}`,
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

    await waitForHealthy(`http://localhost:${ctx.hostPort}`, 120_000);
  });

  test.afterAll(async () => {
    const env = {
      ...process.env,
      COMPOSE_PROJECT_NAME: ctx.projectName,
      CONTAINER_NAME: ctx.containerName,
      HOST_PORT: String(ctx.hostPort),
      DATA_DIR: ctx.dataDir,
      AUTH_TOKENS: ctx.authToken,
    };

    try {
      execFileSyncQuiet('docker', ['compose', 'down', '--remove-orphans'], { cwd: repoRoot, env });
    } finally {
      if (ctx.dataDir) {
        // dataDir is <tmp>/data
        const tmpDir = path.dirname(ctx.dataDir);
        fs.rmSync(tmpDir, { recursive: true, force: true });
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

    const shortCode = new URL(shortURL).pathname.replace(/^\/+/, '').split('/').pop();
    expect(shortCode).toBeTruthy();
    expect(shortCode).toMatch(/^[A-Za-z0-9_-]+$/);

    const dbPath = path.join(ctx.dataDir, 'database.db');

    const deadline = Date.now() + 10_000;
    while (Date.now() < deadline) {
      if (fs.existsSync(dbPath)) {
        const persisted = sqliteGetOriginalURL(dbPath, shortCode);
        if (persisted) {
          expect(persisted).toBe(originalURL);
          return;
        }
      }

      await sleep(250);
    }

    throw new Error(`Row not found in sqlite DB for short_code=${shortCode}`);
  });
});
