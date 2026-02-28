const puppeteer = require('puppeteer');
const fs = require('fs');

const BASE_URL = process.env.BASE_URL || 'https://port-5173-ae2842d.xhd2015.xyz';
const VIEWPORT_WIDTH = parseInt(process.env.VIEWPORT_WIDTH || '375');
const VIEWPORT_HEIGHT = parseInt(process.env.VIEWPORT_HEIGHT || '800');
const HEADLESS = process.env.HEADLESS !== 'false';

(async () => {
    console.log('DEBUG: Starting debug.js');
    // Check if stdin has piped data
    const isPipe = !process.stdin.isTTY;
    let scriptArg;
    
    if (isPipe) {
        console.log('DEBUG: stdin is pipe');
        // Read script from stdin
        scriptArg = fs.readFileSync('/dev/stdin', 'utf-8').trim();
        console.log('DEBUG: script from stdin:', scriptArg);
    } else {
        console.log('DEBUG: stdin is TTY');
        scriptArg = process.argv[2];
    }

    const browser = await puppeteer.launch({
        headless: HEADLESS,
        args: ['--no-sandbox']
    });

    const page = await browser.newPage();
    await page.setViewport({ width: VIEWPORT_WIDTH, height: VIEWPORT_HEIGHT });
    
    // Add waitForTimeout if not exists (Puppeteer compatibility)
    if (!page.waitForTimeout) {
        page.waitForTimeout = function(timeout) {
            return new Promise(resolve => setTimeout(resolve, timeout));
        };
    }

    console.log(`Base URL: ${BASE_URL}`);
    console.log(`Viewport: ${VIEWPORT_WIDTH}x${VIEWPORT_HEIGHT}\n`);

    try {
        if (scriptArg) {
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
        } else {
            // Default: navigate to base URL and show HTML
            await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
            console.log(`Page title: ${await page.title()}`);
            const html = await page.content();
            console.log('\nPage HTML (first 500 chars):');
            console.log(html.substring(0, 500));
        }
    } catch (e) {
        console.error('Error:', e.message);
    }

    await browser.close();
    console.log('\nDone');
    process.exit(0);
})();
