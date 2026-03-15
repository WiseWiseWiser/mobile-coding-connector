// Wait for page to load
await new Promise(r => setTimeout(r, 3000));

// Take screenshot to see current state
const screenshot = await page.screenshot({encoding: "base64"});
console.log("SCREENSHOT:", screenshot);

// Find the opencode project - look for it in the list
const links = await page.$$eval("a", links => links.map(l => ({text: l.textContent?.trim(), href: l.href})));
console.log("Links found:", JSON.stringify(links.slice(0, 20)));

// Navigate to project/opencode
await navigate("/project/opencode", {waitUntil: "networkidle0"});
await new Promise(r => setTimeout(r, 2000));

// Take screenshot
const screenshot2 = await page.screenshot({encoding: "base64"});
console.log("SCREENSHOT2:", screenshot2);

// Look for Worktrees section
const html = await page.content();
console.log("Has Worktrees:", html.includes("Worktrees"));

// Done
