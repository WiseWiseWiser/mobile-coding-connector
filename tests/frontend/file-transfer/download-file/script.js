await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
const row = page.locator('[data-testid="file-transfer-row"], .file-transfer-row').filter({ hasText: 'hello.txt' }).first();
await row.waitFor({ state: 'visible', timeout: 30000 });
const downloadPromise = page.waitForEvent('download');
await row.getByRole('button', { name: /download/i }).click();
const download = await downloadPromise;
console.log(JSON.stringify({
  ok: true,
  downloadedName: download.suggestedFilename(),
}));