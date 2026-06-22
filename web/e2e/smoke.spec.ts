import { test, expect } from '@playwright/test';

test('app boots and the signup page renders', async ({ page }) => {
  await page.goto('/signup');
  await expect(page.getByRole('heading', { name: 'Sign up' })).toBeVisible();
});
