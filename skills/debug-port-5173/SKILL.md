# Debug Port 5173

Run with: `node debug.js [script]` or `echo "script" | node debug.js`

## Quick Examples

```bash
# Get page title
printf "console.log(await page.evaluate(() => document.title))" | node debug.js

# Navigate (auto-prepends BASE_URL)
printf "await navigate('/mockups/path-input'); console.log(await page.title())" | node debug.js

# Get elements
printf "await navigate('/mockups/path-input'); await page.waitForSelector('.path-input-field'); console.log(await page.evaluate(() => document.querySelectorAll('.path-input-field').length))" | node debug.js

# Screenshot
printf "await navigate('/mockups/path-input'); const buf = await page.screenshot(); fs.writeFileSync('screenshot.png', buf)" | node debug.js
```

## Script Variables

| Variable | Description |
|----------|-------------|
| page | Puppeteer Page object |
| browser | Puppeteer Browser object |
| console | Node console |
| fs | Node fs module |
| BASE_URL | Base URL string |
| VIEWPORT_WIDTH | Viewport width |
| VIEWPORT_HEIGHT | Viewport height |
| navigate(url, opts) | Navigate helper (auto-prepends BASE_URL) |

Note: Use `console.log()` to output results. Script runs in async context, use `await`.
