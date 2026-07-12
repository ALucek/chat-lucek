import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const FAKE_GSI = readFileSync(path.join(__dirname, 'fake-gsi.js'), 'utf8');

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

// loginViaMagicLink drives the real request/verify flow using the fake mailer.
async function loginViaMagicLink(
  page: import('@playwright/test').Page,
  email: string,
) {
  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByRole('button', { name: /sign in link/i }).click();
  // Wait for the request to complete so the link is captured before we read it.
  await expect(page.getByText(/check your inbox/i)).toBeVisible();
  const res = await page.request.get(
    `${API}/api/magic/latest?email=${encodeURIComponent(email)}`,
  );
  const { link } = (await res.json()) as { link: string };
  await page.goto(link);
  await expect(page).toHaveURL('/');
}

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

test('a follow-up message appends to the same conversation', async ({
  page,
}) => {
  await loginViaMagicLink(page, `e2e-multi-${Date.now()}@gmail.com`);

  const box = page.getByPlaceholder('Send a message…');
  const send = page.getByRole('button', { name: 'Send' });

  // First message creates the conversation and routes to it.
  await box.fill('First question about cats');
  await send.click();
  await expect(page).toHaveURL(/\/c\/\d+$/);
  const url = page.url();
  await expect(page.getByText('Hello from the stub.')).toHaveCount(1);

  // Send returns once the stream ends; a second turn appends in place.
  await expect(send).toBeVisible();
  await box.fill('Second follow-up question');
  await send.click();
  await expect(page.getByText('Second follow-up question')).toBeVisible();
  await expect(page.getByText('Hello from the stub.')).toHaveCount(2);
  expect(page.url()).toBe(url); // same conversation, no new route
});

test('deleting the open conversation removes it and routes home', async ({
  page,
}) => {
  await loginViaMagicLink(page, `e2e-del-${Date.now()}@gmail.com`);

  // A message creates the conversation we then delete.
  await page
    .getByPlaceholder('Send a message…')
    .fill('Delete me from the sidebar');
  await page.getByRole('button', { name: 'Send' }).click();
  await expect(page).toHaveURL(/\/c\/\d+$/);

  const link = page.getByRole('link', { name: /Delete me from the/i });
  await expect(link).toBeVisible();

  // Open the row's action menu, then confirm the delete.
  await link.hover();
  await page.getByRole('button', { name: 'Conversation actions' }).click();
  await page.getByRole('button', { name: 'Delete' }).click(); // menu item
  await page.getByRole('button', { name: 'Delete' }).click(); // confirm

  // It was the open conversation, so the sidebar row goes and we land home.
  await expect(page).toHaveURL('/');
  await expect(link).toHaveCount(0);
});
