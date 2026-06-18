await page.goto(BASE_URL + '/home/file-transfer', { waitUntil: 'domcontentloaded' });
await page.waitForSelector('h2', { timeout: 30000 });
const emptyStateVisible = (await page.getByText(/no files yet/i).count()) > 0;
const rows = await page.locator('[data-testid="file-transfer-row"], .file-transfer-row').count();
console.log(JSON.stringify({
  ok: emptyStateVisible && rows === 0,
  emptyStateVisible,
  fileCount: rows,
}));