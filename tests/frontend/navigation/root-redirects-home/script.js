await page.goto(BASE_URL + '/', { waitUntil: 'domcontentloaded' });
await page.waitForURL('**/home**', { timeout: 30000 });
const url = page.url();
console.log(JSON.stringify({
  ok: url.endsWith('/home') || url.includes('/home'),
  url,
}));