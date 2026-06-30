const serviceName = 'ui-disable-running';

await page.goto(BASE_URL + '/home/service', { waitUntil: 'domcontentloaded' });
const card = page.locator('.mcc-service-card').filter({ hasText: serviceName }).first();
await card.waitFor({ state: 'visible', timeout: 60000 });

const disableBtn = card.getByRole('button', { name: /^disable$/i });
await disableBtn.waitFor({ state: 'visible', timeout: 30000 });
await disableBtn.click();

const modal = page.locator('.mcc-modal');
await modal.waitFor({ state: 'visible', timeout: 30000 });
const modalText = ((await modal.textContent()) ?? '').trim();

const confirmBtn = modal.getByRole('button', { name: /disable/i });
await confirmBtn.click();
await modal.waitFor({ state: 'hidden', timeout: 30000 }).catch(() => {});

await page.waitForTimeout(1500);

const statusBadge = card.locator('.mcc-service-status-badge');
const statusText = ((await statusBadge.textContent()) ?? '').trim();
const pidLine = ((await card.locator('.mcc-service-meta').textContent()) ?? '').trim();

const servicesResp = await page.request.get(BASE_URL + '/api/services');
const services = await servicesResp.json();
const svc = Array.isArray(services) ? services.find((s) => s.name === serviceName) : null;
const apiPid = svc?.pid ?? 0;
const apiStatus = svc?.status ?? '';
const apiEnabled = svc?.enabled;
const stillRunning = apiPid > 0 && (apiStatus === 'running' || apiStatus === 'starting');

console.log(JSON.stringify({
  ok: modalText.toLowerCase().includes("won't stop immediately") && stillRunning,
  modalText,
  statusText,
  pidLine,
  apiPid,
  apiStatus,
  apiEnabled,
  stillRunning,
}));