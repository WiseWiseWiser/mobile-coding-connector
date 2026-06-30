const expectedContent = 'seeded-scratch-content-for-display';
await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
const textarea = page.locator('[data-testid="file-transfer-scratch-input"]');
await textarea.waitFor({ state: 'visible', timeout: 30000 });
const textareaValue = await textarea.inputValue();
console.log(JSON.stringify({
  ok: textareaValue === expectedContent,
  textareaValue,
  expectedContent,
}));