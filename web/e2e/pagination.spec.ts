import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { execSync } from 'node:child_process';
import path from 'node:path';

const FAKE_GSI = readFileSync(path.join(__dirname, 'fake-gsi.js'), 'utf8');
const REPO_ROOT = path.join(__dirname, '..', '..');

// psql pipes SQL into the compose db container over stdin (no -c quoting).
function psql(sql: string) {
  execSync(
    `docker compose exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'`,
    { cwd: REPO_ROOT, input: sql },
  );
}

async function login(page: import('@playwright/test').Page, email: string) {
  await page.route('https://accounts.google.com/gsi/client', (route) =>
    route.fulfill({ contentType: 'application/javascript', body: FAKE_GSI }),
  );
  await page.addInitScript((e) => {
    (window as unknown as { __E2E_EMAIL__: string }).__E2E_EMAIL__ = e;
  }, email);
  await page.goto('/login');
  await page.getByRole('button', { name: 'Sign in with Google' }).click();
  await expect(page).toHaveURL('/');
}

test('messages: newest page first, older loads on scroll without jumping', async ({
  page,
}) => {
  await login(page, `e2e-page-${Date.now()}@gmail.com`);
  await page.getByRole('button', { name: 'New conversation' }).click();
  await expect(page).toHaveURL(/\/c\/\d+$/);
  const cid = page.url().split('/c/')[1];

  psql(
    `insert into messages (conversation_id, role, content) select ${cid}, 'user', 'msg-'||lpad(g::text,3,'0') from generate_series(1,60) g;`,
  );
  await page.reload();

  // Newest page only: msg-060 rendered, msg-001 not yet loaded.
  await expect(page.getByText('msg-060', { exact: true })).toBeVisible();
  await expect(page.getByText('msg-001', { exact: true })).toHaveCount(0);

  // Jump to the top and capture the anchor's position in the same frame.
  const scroller = page.getByTestId('messages-scroll');
  const beforeY = await scroller.evaluate((el) => {
    el.scrollTop = 0;
    const li = [...el.querySelectorAll('li')].find((n) =>
      n.textContent?.includes('msg-011'),
    );
    return li ? li.getBoundingClientRect().y : null;
  });

  // Older page loads in.
  await expect(page.getByText('msg-001', { exact: true })).toHaveCount(1);

  // The viewport stays anchored: msg-011 keeps its position. A broken restore
  // would jump it by ~10 message-heights (~800px); the small tolerance covers
  // the stick-to-bottom resize spring settling.
  expect(beforeY).not.toBeNull();
  await page.waitForTimeout(500);
  const afterY = await page
    .getByText('msg-011', { exact: true })
    .evaluate((n) => n.getBoundingClientRect().y);
  expect(Math.abs(afterY - (beforeY as number))).toBeLessThan(80);
});

test('sidebar: loads more conversations on scroll', async ({ page }) => {
  await login(page, `e2e-convos-${Date.now()}@gmail.com`);
  // One real conversation gives us the user id to attach the rest to.
  await page.getByRole('button', { name: 'New conversation' }).click();
  await expect(page).toHaveURL(/\/c\/\d+$/);

  // Seed 40 more titled conversations for this user (page size is 30).
  psql(
    `insert into conversations (user_id, title) select (select user_id from conversations order by id desc limit 1), 'convo-'||lpad(g::text,3,'0') from generate_series(1,40) g;`,
  );
  await page.goto('/');

  const sidebar = page.getByRole('navigation');
  // convo-040 is newest activity, on the first page; convo-001 is not yet loaded.
  await expect(page.getByText('convo-040', { exact: true })).toBeVisible();
  await expect(page.getByText('convo-001', { exact: true })).toHaveCount(0);

  await sidebar.evaluate((el) => (el.scrollTop = el.scrollHeight));
  await expect(page.getByText('convo-001', { exact: true })).toHaveCount(1);
});
