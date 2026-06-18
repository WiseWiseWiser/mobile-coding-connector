await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
const row = page.locator('[data-testid="file-transfer-row"], .file-transfer-row').filter({ hasText: 'temp.txt' }).first();
await row.waitFor({ state: 'visible', timeout: 30000 });
const beforeCount = await page.locator('[data-testid="file-transfer-row"], .file-transfer-row').count();
page.once('dialog', async (dialog) => { await dialog.accept(); });
await row.getByRole('button', { name: /remove/i }).click();
await page.locator('[data-testid="file-transfer-row"], .file-transfer-row').filter({ hasText: 'temp.txt' }).waitFor({ state: 'detached', timeout: 30000 });
const afterCount = await page.locator('[data-testid="file-transfer-row"], .file-transfer-row').count();
const tempVisible = (await page.locator('[data-testid="file-transfer-row"], .file-transfer-row').filter({ hasText: 'temp.txt' }).count()) > 0;
console.log(JSON.stringify({
  ok: !tempVisible && afterCount < beforeCount,
  beforeCount,
  afterCount,
  tempVisible,
}));