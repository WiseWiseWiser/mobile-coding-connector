await page.goto(BASE_URL + '/home/settings', { waitUntil: 'domcontentloaded' });
await page.waitForSelector('h2', { timeout: 30000 });
const heading = (await page.locator('h2').first().textContent())?.trim() ?? '';
console.log(JSON.stringify({
  ok: heading === 'Settings',
  url: page.url(),
  heading,
}));