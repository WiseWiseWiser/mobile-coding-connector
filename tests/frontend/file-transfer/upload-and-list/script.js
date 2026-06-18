await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
await page.waitForSelector('h2', { timeout: 30000 });
const fixturePath = CASE_DIR + '/testdata/sample.txt';
const fileInput = page.locator('input[type="file"]');
await fileInput.setInputFiles(fixturePath);
const row = page.locator('[data-testid="file-transfer-row"], .file-transfer-row').filter({ hasText: 'sample.txt' }).first();
await row.waitFor({ state: 'visible', timeout: 60000 });
const rowText = (await row.textContent())?.trim() ?? '';
const fileCount = await page.locator('[data-testid="file-transfer-row"], .file-transfer-row').count();
console.log(JSON.stringify({
  ok: fileCount >= 1 && rowText.includes('sample.txt'),
  fileName: 'sample.txt',
  fileCount,
  rowText,
}));