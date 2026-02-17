# Quick Test: OpenCode Settings

A quick testing skill using Puppeteer to debug and verify the OpenCode settings page functionality.

## Purpose

This skill tests the OpenCode settings page at `/project/mobile-coding-connector/agent/opencode/settings` to verify:

1. **Port Configuration Display** - The web server port field is displayed correctly
2. **Port Modification** - Users can change the port value
3. **Save Functionality** - Changes can be saved
4. **Correct Port Usage** - Server uses the configured port (not random)

## Usage

### Prerequisites

Ensure you have the dependencies installed:

```bash
cd /root/mobile-coding-connector/skills/quick-test-opencode
npm install
```

### Run the Test

```bash
# Run in headless mode (default)
npm test

# Run with visible browser window
npm run test:headed

# Run with debug output
npm run test:debug
```

### Environment Variables

- `TEST_URL` - The base URL to test (default: `https://agent-fast-apex-nest-23aed.xhd2015.xyz`)
- `HEADLESS` - Run browser in headless mode (default: `true`)
- `DEBUG` - Enable debug output (default: `false`)

Example:
```bash
TEST_URL=http://localhost:23712 npm test
```

## Test Output

The test will:

1. Navigate to the settings page
2. Take screenshots at each step
3. Check for the port input field
4. Attempt to change the port value
5. Look for the Save button
6. Generate a report with screenshots

Screenshots are saved to `/tmp/opencode-test-screenshots/`

## Debugging Tips

1. **Port input not found**: The test will show all input elements found on the page
2. **Screenshots are blank**: Check if authentication is required
3. **Connection refused**: Verify the TEST_URL is correct and accessible

## Implementation Details

The test uses:
- **Puppeteer**: For browser automation
- **Headless Chrome**: For running tests without GUI
- **Screenshots**: For visual debugging
- **Element inspection**: To verify form fields

## Related Files

- `test-opencode-settings.js` - Main test script
- `package.json` - Dependencies and scripts
- `/root/mobile-coding-connector/server/agents/opencode/` - Backend implementation
- `/root/mobile-coding-connector/ai-critic-react/src/v2/mcc/agent/OpencodeSettings.tsx` - Frontend component
