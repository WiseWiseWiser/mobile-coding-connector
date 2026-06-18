await page.goto(BASE_URL + '/home/tools', { waitUntil: 'domcontentloaded' });
await page.waitForSelector('h2', { timeout: 60000 });
const heading = (await page.locator('h2').first().textContent())?.trim() ?? '';
let foundationVisible = false;
try {
  await page.getByText('Foundation', { exact: true }).first().waitFor({ timeout: 60000 });
  foundationVisible = true;
} catch {
  foundationVisible = false;
}
console.log(JSON.stringify({
  ok: heading.includes('Server Tools') || foundationVisible,
  url: page.url(),
  heading,
  foundationVisible,
}));