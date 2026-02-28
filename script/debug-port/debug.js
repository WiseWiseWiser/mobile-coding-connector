const { chromium } = require('playwright');
const fs = require('fs');

const BASE_URL = process.env.BASE_URL || 'https://port-5173-ae2842d.xhd2015.xyz';
const VIEWPORT_WIDTH = parseInt(process.env.VIEWPORT_WIDTH || '375');
const VIEWPORT_HEIGHT = parseInt(process.env.VIEWPORT_HEIGHT || '800');
const HEADLESS = process.env.HEADLESS !== 'false';

(async () => {
    console.log('DEBUG: Starting debug.js');
    const scriptArg = process.argv[2];
    if (!scriptArg || !scriptArg.trim()) {
        console.error('Error: missing script argument');
        process.exit(1);
    }
    console.log('DEBUG: script from argv:', scriptArg);

    let browser;
    try {
        browser = await chromium.launch({
            headless: HEADLESS,
            args: ['--no-sandbox']
        });
    } catch (e) {
        console.error('Failed to launch Playwright browser:', e.message);
        console.error('Run `cd script/debug-port && npx playwright install` to install Playwright browsers.');
        process.exit(1);
    }

    const context = await browser.newContext({
        viewport: { width: VIEWPORT_WIDTH, height: VIEWPORT_HEIGHT }
    });
    const page = await context.newPage();

    console.log(`Base URL: ${BASE_URL}`);
    console.log(`Viewport: ${VIEWPORT_WIDTH}x${VIEWPORT_HEIGHT}\n`);

    try {
        console.log(`Running script:\n${scriptArg}\n`);
        
        // Use Function constructor with named async function
        const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
        
        // Pass navigate as part of context object
        const ctx = { 
            page, 
            browser, 
            console, 
            fs, 
            BASE_URL, 
            VIEWPORT_WIDTH, 
            VIEWPORT_HEIGHT,
            navigate: async (url, options) => {
                const fullUrl = url.startsWith('http') ? url : BASE_URL + url;
                return await page.goto(fullUrl, options);
            }
        };
        
        // Use with to give script access to context
        const fn = new AsyncFunction('ctx', `
            const { page, browser, console, fs, BASE_URL, navigate, VIEWPORT_WIDTH, VIEWPORT_HEIGHT } = ctx;
            ${scriptArg}
        `);
        
        await fn(ctx);
    } catch (e) {
        console.error('Error:', e.message);
    }

    await browser.close();
    console.log('\nDone');
    process.exit(0);
})();
