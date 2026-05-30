const { chromium } = require('playwright');

const BASE_URL = process.env.BASE_URL || 'http://localhost:3580';
const HEADLESS = process.env.HEADLESS !== 'false';
const MAX_ITERATIONS = parseInt(process.env.MAX_ITERATIONS || '0', 10);

// ---- Helpers ----

function randomDelay(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

function ts() {
    return new Date().toISOString();
}

// ---- Check for duplicate terminal sessions ----

async function checkSessions(page) {
    const result = await page.evaluate(async (baseUrl) => {
        try {
            const res = await fetch(`${baseUrl}/api/terminal/sessions`, {
                headers: { 'X-Fuzzy-Check': '1' }
            });
            if (!res.ok) {
                return { error: `HTTP ${res.status}` };
            }
            const data = await res.json();
            return { sessions: data.sessions || [] };
        } catch (e) {
            return { error: e.message };
        }
    }, BASE_URL);

    if (result.error) {
        console.log(`[${ts()}] [CHECK] Error: ${result.error}`);
        return { count: 0, duplicate: false };
    }

    const sessions = result.sessions;
    const count = sessions.length;
    const duplicate = count > 1;
    const ids = sessions.map(s => s.id).join(', ') || '(none)';

    console.log(`[${ts()}] [CHECK] Sessions: ${count} [${ids}]${duplicate ? ' *** DUPLICATE ***' : ''}`);
    return { count, duplicate, sessions };
}

// ---- Latency injection (no app code changes needed) ----

async function injectLatency(page) {
    // 1. Random delay on GET /api/terminal/sessions
    await page.route('**/api/terminal/sessions**', async (route) => {
        const headers = route.request().headers();
        // Skip our own check requests (marked with X-Fuzzy-Check header)
        if (headers['x-fuzzy-check']) {
            await route.continue();
            return;
        }
        const delay = randomDelay(100, 3000);
        console.log(`[${ts()}] [LATENCY] Delaying sessions API by ${delay}ms`);
        await new Promise(r => setTimeout(r, delay));
        await route.continue();
    });

    // 2. Log all WebSocket creation in the browser so we can see
    //    if the frontend opens more WS connections than expected.
    await page.addInitScript(() => {
        const OrigWebSocket = window.WebSocket;
        window.WebSocket = function (...args) {
            console.log('[WS-FUZZY] new WebSocket:', args[0]);
            return new OrigWebSocket(...args);
        };
        window.WebSocket.prototype = OrigWebSocket.prototype;
        Object.assign(window.WebSocket, OrigWebSocket);
    });

    // 3. Listen for browser console messages (so WS-FUZZY logs show up)
    page.on('console', (msg) => {
        const text = msg.text();
        if (text.startsWith('[WS-FUZZY]')) {
            console.log(`[${ts()}] ${text}`);
        }
    });
}

// ---- Main loop ----

async function main() {
    console.log(`[${ts()}] [FUZZY] === Terminal Duplicate Session Fuzzy Test ===`);
    console.log(`[${ts()}] [FUZZY] BASE_URL:     ${BASE_URL}`);
    console.log(`[${ts()}] [FUZZY] HEADLESS:     ${HEADLESS}`);
    console.log(`[${ts()}] [FUZZY] MAX_ITER:     ${MAX_ITERATIONS} (0=forever)`);
    console.log(`[${ts()}] [FUZZY] Press Ctrl-C to stop\n`);

    const browser = await chromium.launch({
        headless: HEADLESS,
        args: ['--no-sandbox']
    });

    const context = await browser.newContext({
        viewport: { width: 375, height: 800 }
    });
    const page = await context.newPage();

    await injectLatency(page);

    let iteration = 0;
    let duplicateCount = 0;
    let totalChecks = 0;
    let maxSessionsSeen = 0;
    const startTime = Date.now();

    const loop = () => {
        if (MAX_ITERATIONS > 0 && iteration >= MAX_ITERATIONS) {
            return false;
        }
        iteration++;
        return true;
    };

    try {
        while (loop()) {
            const actionDelay = randomDelay(500, 5000);
            console.log(`\n[${ts()}] [FUZZY] === Iteration ${iteration} (delay ${actionDelay}ms) ===`);

            // ---- Navigate ----
            console.log(`[${ts()}] [FUZZY] Navigating to /terminal...`);
            try {
                await page.goto(`${BASE_URL}/terminal`, {
                    waitUntil: 'domcontentloaded',
                    timeout: 20000
                });
            } catch (e) {
                console.log(`[${ts()}] [FUZZY] Navigation failed: ${e.message}`);
                continue;
            }

            const initWait = randomDelay(2000, 5000);
            await new Promise(r => setTimeout(r, initWait));

            totalChecks++;
            const r1 = await checkSessions(page);
            if (r1.count > maxSessionsSeen) maxSessionsSeen = r1.count;
            if (r1.duplicate) duplicateCount++;

            // ---- Refresh ----
            const refreshDelay = randomDelay(1000, 5000);
            console.log(`[${ts()}] [FUZZY] Refreshing in ${refreshDelay}ms...`);
            await new Promise(r => setTimeout(r, refreshDelay));

            try {
                await page.reload({
                    waitUntil: 'domcontentloaded',
                    timeout: 20000
                });
            } catch (e) {
                console.log(`[${ts()}] [FUZZY] Reload failed: ${e.message}`);
                continue;
            }

            const postRefreshWait = randomDelay(2000, 5000);
            await new Promise(r => setTimeout(r, postRefreshWait));

            totalChecks++;
            const r2 = await checkSessions(page);
            if (r2.count > maxSessionsSeen) maxSessionsSeen = r2.count;
            if (r2.duplicate) duplicateCount++;

            // ---- Stats ----
            const elapsed = Math.round((Date.now() - startTime) / 1000);
            console.log(`[${ts()}] [STATS] iter=${iteration} checks=${totalChecks} dup=${duplicateCount} maxSessions=${maxSessionsSeen} elapsed=${elapsed}s`);
        }
    } finally {
        await browser.close();
    }

    const elapsed = Math.round((Date.now() - startTime) / 1000);
    console.log(`\n[${ts()}] [FUZZY] ========== Final Summary ==========`);
    console.log(`[${ts()}] [FUZZY] Iterations:        ${iteration}`);
    console.log(`[${ts()}] [FUZZY] Checks:            ${totalChecks}`);
    console.log(`[${ts()}] [FUZZY] Duplicates found:  ${duplicateCount}`);
    console.log(`[${ts()}] [FUZZY] Max sessions seen: ${maxSessionsSeen}`);
    console.log(`[${ts()}] [FUZZY] Elapsed:           ${elapsed}s`);
    console.log(`[${ts()}] [FUZZY] Result: ${duplicateCount === 0 ? 'PASSED - No duplicates' : 'FAILED - Duplicates detected!'}`);

    if (duplicateCount > 0) {
        process.exit(1);
    }
}

main().catch(err => {
    console.error(`[${ts()}] [FUZZY] Fatal:`, err.message);
    process.exit(1);
});
