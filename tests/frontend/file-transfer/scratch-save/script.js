const saveText = 'saved-from-playwright-scratch-test';
await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
const textarea = page.locator('[data-testid="file-transfer-scratch-input"]');
await textarea.waitFor({ state: 'visible', timeout: 30000 });
await textarea.fill(saveText);
await page.locator('[data-testid="file-transfer-scratch-save"]').click();
await page.waitForTimeout(1000);
const textareaValue = await textarea.inputValue();
console.log(JSON.stringify({
  ok: textareaValue === saveText,
  saveText,
  textareaValue,
}));