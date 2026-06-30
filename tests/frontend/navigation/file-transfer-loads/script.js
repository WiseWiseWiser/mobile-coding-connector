await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
await page.waitForSelector('h2', { timeout: 30000 });
const heading = (await page.locator('h2').first().textContent())?.trim() ?? '';
const uploadByTestId = (await page.locator('[data-testid="file-transfer-upload"]').count()) > 0;
const uploadByClass = (await page.locator('.file-transfer-upload').count()) > 0;
const uploadByButton = (await page.getByRole('button', { name: /upload/i }).count()) > 0;
const uploadAreaVisible = uploadByTestId || uploadByClass || uploadByButton;
const scratchAreaVisible = (await page.locator('[data-testid="file-transfer-scratch"]').count()) > 0;
console.log(JSON.stringify({
  ok: heading.includes('File Transfer') && scratchAreaVisible && uploadAreaVisible,
  url: page.url(),
  heading,
  scratchAreaVisible,
  uploadAreaVisible,
}));