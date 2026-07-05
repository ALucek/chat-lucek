import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const FAKE_GSI = readFileSync(path.join(__dirname, 'fake-gsi.js'), 'utf8');

test('sign in with Google, send a message, stream a reply, persist on reload', async ({
  page,
}) => {
  const email = `e2e-${Date.now()}@gmail.com`;
  const message = 'Tell me a joke about cats please';

  // Serve the fake GSI script in place of Google's, and pick this run's email.
  await page.route('https://accounts.google.com/gsi/client', (route) =>
    route.fulfill({ contentType: 'application/javascript', body: FAKE_GSI }),
  );
  await page.addInitScript((e) => {
    (window as unknown as { __E2E_EMAIL__: string }).__E2E_EMAIL__ = e;
  }, email);

  // Real login flow: button → callback → loginWithGoogle → /api/google → session.
  await page.goto('/login');
  await page.getByRole('button', { name: 'Sign in with Google' }).click();
  await expect(page).toHaveURL('/');

  // Draft new chat: sending creates the conversation, routes, and streams.
  await page.getByPlaceholder('Send a message…').fill(message);
  await page.getByRole('button', { name: 'Send' }).click();
  await expect(page).toHaveURL(/\/c\/\d+$/);
  await expect(page.getByText('Hello from the stub.')).toBeVisible();

  // The run timeline renders: a reasoning step and a folded subagent.
  await expect(page.getByText('thinking')).toBeVisible();
  const subagent = page.getByRole('button').filter({ hasText: 'subagent' });
  await expect(subagent).toBeVisible();

  // The subagent is folded: its nested search + summary appear only on expand.
  await expect(page.getByText('search', { exact: true })).toHaveCount(0);
  await subagent.click();
  await expect(page.getByText('search', { exact: true })).toBeVisible();
  await expect(page.getByText('subagent summary')).toBeVisible();

  // The first message names the conversation (first five words) in the sidebar.
  await expect(
    page.getByRole('link', { name: 'Tell me a joke about' }),
  ).toBeVisible();

  // Reload — the persisted trace rehydrates the timeline (real DB).
  await page.reload();
  await expect(page.getByText(message)).toBeVisible();
  await expect(page.getByText('Hello from the stub.')).toBeVisible();
  await expect(page.getByText('thinking')).toBeVisible();
  await expect(
    page.getByRole('button').filter({ hasText: 'subagent' }),
  ).toBeVisible();
});

test('composer grows with content up to a max, then stops', async ({
  page,
}) => {
  const email = `e2e-grow-${Date.now()}@gmail.com`;

  await page.route('https://accounts.google.com/gsi/client', (route) =>
    route.fulfill({ contentType: 'application/javascript', body: FAKE_GSI }),
  );
  await page.addInitScript((e) => {
    (window as unknown as { __E2E_EMAIL__: string }).__E2E_EMAIL__ = e;
  }, email);

  await page.goto('/login');
  await page.getByRole('button', { name: 'Sign in with Google' }).click();
  await expect(page).toHaveURL('/');

  const box = page.getByPlaceholder('Send a message…');
  const send = page.getByRole('button', { name: 'Send' });
  const height = async () => (await box.boundingBox())!.height;

  await box.fill('one line');
  const single = await height();

  // One line: input and Send are vertically centered on each other.
  let b = (await box.boundingBox())!;
  let s = (await send.boundingBox())!;
  expect(
    Math.abs(b.y + b.height / 2 - (s.y + s.height / 2)),
  ).toBeLessThanOrEqual(2);

  await box.fill(Array.from({ length: 4 }, (_, i) => `line ${i}`).join('\n'));
  expect(await height()).toBeGreaterThan(single);

  // Grown: Send sticks to the bottom of the (taller) input.
  b = (await box.boundingBox())!;
  s = (await send.boundingBox())!;
  expect(Math.abs(b.y + b.height - (s.y + s.height))).toBeLessThanOrEqual(2);

  await box.fill(Array.from({ length: 40 }, (_, i) => `line ${i}`).join('\n'));
  const capped = await height();
  expect(capped).toBeLessThanOrEqual(162); // max-h-40 (160px) + borders
  expect(capped).toBeGreaterThan(single);
});
