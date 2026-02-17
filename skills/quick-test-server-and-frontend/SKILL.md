# Quick Test Skill: Server and Frontend

## Overview

This skill provides automated testing for the server and frontend integration using Puppeteer. It can be used to verify various pages including the OpenCode settings page.

## Quick Start

```bash
cd /root/mobile-coding-connector/skills/quick-test-server-and-frontend
npm install
npm test
```

## What It Tests

1. **Port Configuration Display** - Verifies the port field is visible
2. **Port Modification** - Tests changing the port value  
3. **Save Functionality** - Checks for save button
4. **Port Persistence** - Verifies configured port is used (not random)

## Environment Variables

- `TEST_URL` - Base URL (default: `https://agent-fast-apex-nest-23aed.xhd2015.xyz`)
- `HEADLESS` - Run headless (default: `true`)
- `DEBUG` - Enable debug output (default: `false`)

## Output

- Screenshots saved to `/tmp/opencode-test-screenshots/`
- Console output with test results
- Detailed error messages if tests fail

## Related Code

- Backend: `/root/mobile-coding-connector/server/agents/opencode/`
- Frontend: `/root/mobile-critic-react/src/v2/mcc/agent/OpencodeSettings.tsx`
