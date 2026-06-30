await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
await page.waitForSelector('.mcc-setup', { timeout: 30000 });

const setupVisible = (await page.locator('.mcc-setup').count()) > 0;
const genBtn = page.locator('button:has-text("Generate Random")');
await genBtn.first().click();
await page.waitForTimeout(1500);

const errorText = (await page.locator('.mcc-setup-error').textContent().catch(() => ''))?.trim() ?? '';
const credential = await page.locator('.mcc-setup-credential-input').inputValue();

console.log(JSON.stringify({
  ok: setupVisible && errorText === '' && credential.length === 64,
  setupVisible,
  errorText,
  credentialLength: credential.length,
  url: page.url(),
}));