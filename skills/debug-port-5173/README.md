# debug-port-5173

Debug skill for testing pages on port 5173 using Puppeteer.

## Usage

```bash
# Run with node directly
node debug.js [script]

# Or pipe script from stdin
echo "script" | node debug.js
```

## Examples

```bash
# Default: show HTML of base URL
node debug.js

# Get page title
printf "console.log(await page.evaluate(() => document.title))" | node debug.js

# Navigate using navigate() helper (auto-prepends BASE_URL)
printf "await navigate('/mockups/path-input'); console.log(await page.title())" | node debug.js

# Get elements after navigation
printf "await navigate('/mockups/path-input'); await page.waitForSelector('.path-input-field'); console.log(await page.evaluate(() => document.querySelectorAll('.path-input-field').length))" | node debug.js

# Take screenshot
printf "await navigate('/mockups/path-input'); const buf = await page.screenshot(); fs.writeFileSync('screenshot.png', buf); console.log('Saved')" | node debug.js

# Full example with element data
printf "await navigate('/mockups/path-input'); await page.waitForSelector('.path-input-field'); const inputs = await page.evaluate(() => Array.from(document.querySelectorAll('.path-input-field')).map(el => ({height: el.offsetHeight, value: el.value}))); console.log(JSON.stringify(inputs, null, 2))" | node debug.js
```

## Available Variables in Script

- `page` - Puppeteer Page object
- `browser` - Puppeteer Browser object  
- `console` - Node console
- `fs` - Node fs module
- `BASE_URL` - Base URL string
- `VIEWPORT_WIDTH` - Viewport width
- `VIEWPORT_HEIGHT` - Viewport height
- `navigate(url, options)` - Helper to navigate (auto-prepends BASE_URL)

## Environment Variables

```bash
BASE_URL=https://example.com VIEWPORT_WIDTH=414 node debug.js "script"
```

- `BASE_URL` - Base URL (default: https://port-5173-ae2842d.xhd2015.xyz)
- `VIEWPORT_WIDTH` - Viewport width (default: 375)
- `VIEWPORT_HEIGHT` - Viewport height (default: 800)
- `HEADLESS` - Run in headless mode (default: true)
