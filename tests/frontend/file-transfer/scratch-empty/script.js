await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
await page.waitForSelector('[data-testid="file-transfer-scratch"]', { timeout: 30000 });
const scratchAreaVisible = (await page.locator('[data-testid="file-transfer-scratch"]').count()) > 0;
const textarea = page.locator('[data-testid="file-transfer-scratch-input"]');
await textarea.waitFor({ state: 'visible', timeout: 30000 });
const textareaValue = await textarea.inputValue();
const textareaEmpty = textareaValue === '';
console.log(JSON.stringify({
  ok: scratchAreaVisible && textareaEmpty,
  scratchAreaVisible,
  textareaEmpty,
  textareaValue,
}));