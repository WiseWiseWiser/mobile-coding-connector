const serviceName = 'ui-enable-stopped';

await page.goto(BASE_URL + '/home/service', { waitUntil: 'domcontentloaded' });
const card = page.locator('.mcc-service-card').filter({ hasText: serviceName }).first();
await card.waitFor({ state: 'visible', timeout: 60000 });

const enableBtn = card.getByRole('button', { name: /^enable$/i });
await enableBtn.waitFor({ state: 'visible', timeout: 30000 });
await enableBtn.click();

const modal = page.locator('.mcc-modal');
await modal.waitFor({ state: 'visible', timeout: 30000 });
const modalText = ((await modal.textContent()) ?? '').trim();

const confirmBtn = modal.getByRole('button', { name: /enable/i });
await confirmBtn.click();
await modal.waitFor({ state: 'hidden', timeout: 30000 }).catch(() => {});

await page.waitForTimeout(1500);

const disabledBadge = card.locator('.mcc-service-status-badge--disabled, .mcc-service-enabled-badge--disabled');
const disableBtn = card.getByRole('button', { name: /^disable$/i });
const enabledUi = (await disableBtn.count()) > 0 || (await disabledBadge.count()) === 0;

const servicesResp = await page.request.get(BASE_URL + '/api/services');
const services = await servicesResp.json();
const svc = Array.isArray(services) ? services.find((s) => s.name === serviceName) : null;
const apiEnabled = svc?.enabled;
const apiStatus = svc?.status ?? '';

console.log(JSON.stringify({
  ok: (modalText.toLowerCase().includes('daemon') && modalText.toLowerCase().includes('next')) && enabledUi,
  modalText,
  enabledUi,
  apiEnabled,
  apiStatus,
}));