await page.goto(BASE_URL + '/home', { waitUntil: 'domcontentloaded' });
await page.waitForSelector('.mcc-workspace-list', { timeout: 30000 });
const heading = (await page.locator('h2').first().textContent())?.trim() ?? '';
const hasWorkspaceUI = (await page.locator('.mcc-workspace-list').count()) > 0;
const bodyText = await page.locator('body').innerText();
console.log(JSON.stringify({
  ok: hasWorkspaceUI && bodyText.length > 0,
  url: page.url(),
  title: await page.title(),
  heading,
}));