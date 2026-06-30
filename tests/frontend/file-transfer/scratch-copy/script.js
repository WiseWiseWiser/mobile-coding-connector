const expectedContent = 'seeded-scratch-for-copy-test';
await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
const textarea = page.locator('[data-testid="file-transfer-scratch-input"]');
await textarea.waitFor({ state: 'visible', timeout: 30000 });
await page.locator('[data-testid="file-transfer-scratch-copy"]').click();
const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
console.log(JSON.stringify({
  ok: clipboardText === expectedContent,
  clipboardText,
  expectedContent,
}));