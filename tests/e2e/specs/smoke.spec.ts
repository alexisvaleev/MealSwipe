import { test, expect } from '@playwright/test';

// Critical happy path: register → skip onboarding → swipe → generate menu →
// open shopping list → reload keeps session.

const username = `e2e_${Date.now().toString(36)}`;
const password = 'e2e-password-123';

test.describe('MealSwipe smoke', () => {
  test('register → swipe → menu → shopping → reload persists', async ({ page }) => {
    await page.goto('/');

    // --- register ---
    await page.getByRole('link', { name: /начать подбирать|начать/i }).first().click();
    await page.waitForURL(/register|login/);
    if (page.url().includes('login')) {
      await page.getByRole('link', { name: /регистрац|создать/i }).first().click();
    }
    await page.getByLabel(/зовут|имя/i).fill('E2E User');
    await page.getByLabel(/логин/i).fill(username);
    await page.getByLabel(/пароль/i).first().fill(password);
    await page.getByRole('button', { name: /зарегистр/i }).click();

    // --- onboarding: skip if offered ---
    await page.waitForTimeout(1500);
    const skip = page.getByRole('link', { name: /пропустить/i })
      .or(page.getByRole('button', { name: /пропустить/i }));
    if (await skip.count()) await skip.first().click();

    // --- swipe deck ---
    await page.goto('/swipe');
    await expect(page.locator('body')).not.toContainText(/загрузка/i, { timeout: 10_000 });
    const likeBtn = page.getByRole('button', { name: /нравится|лайк|like|♥/i }).first();
    if (await likeBtn.count()) {
      await likeBtn.click();
      await page.waitForTimeout(400);
      await likeBtn.click().catch(() => {});
    }

    // --- menu ---
    await page.goto('/menu');
    const gen = page.getByRole('button', { name: /сгенерировать|создать меню/i }).first();
    if (await gen.count()) await gen.click();
    await page.waitForTimeout(1500);

    // --- shopping ---
    await page.goto('/shopping');
    await page.waitForTimeout(1000);

    // --- session survives reload ---
    await page.reload();
    await expect(page).not.toHaveURL(/login/);
  });
});
