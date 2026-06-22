import { test, expect } from '@playwright/test';

test('signup, send a message, stream a reply, persist on reload', async ({
  page,
}) => {
  const email = `e2e-${Date.now()}@example.com`;
  const message = 'Tell me a joke about cats please';

  // Sign up → lands on the app shell.
  await page.goto('/signup');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill('password123');
  await page.getByRole('button', { name: 'Sign up' }).click();
  await expect(page).toHaveURL('/');

  // Create a conversation.
  await page.getByRole('button', { name: 'New conversation' }).click();
  await expect(page).toHaveURL(/\/c\/\d+$/);

  // Send a message; the stubbed reply streams in.
  await page.getByPlaceholder('Send a message…').fill(message);
  await page.getByRole('button', { name: 'Send' }).click();
  await expect(page.getByText('Hello from the stub.')).toBeVisible();

  // The first message names the conversation (first five words) in the sidebar.
  await expect(
    page.getByRole('link', { name: 'Tell me a joke about' }),
  ).toBeVisible();

  // Reload — the persisted history (real DB) is still there.
  await page.reload();
  await expect(page.getByText(message)).toBeVisible();
  await expect(page.getByText('Hello from the stub.')).toBeVisible();
});
