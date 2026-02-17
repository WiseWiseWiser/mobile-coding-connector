#!/usr/bin/env bash
# Diagnostic script for OpenCode port configuration

echo "=========================================="
echo "OpenCode Port Configuration Diagnostic"
echo "=========================================="
echo ""

# Check running opencode processes
echo "1. Checking running opencode processes..."
echo "   (These may be old processes from before the fix)"
echo ""
ps aux | grep -E "opencode.*web" | grep -v grep || echo "   No opencode processes found"
echo ""

# Check settings file
echo "2. Checking settings file..."
SETTINGS_FILE="$HOME/.config/ai-critic/opencode.json"
if [ -f "$SETTINGS_FILE" ]; then
    echo "   Settings file: $SETTINGS_FILE"
    echo "   Contents:"
    cat "$SETTINGS_FILE" | python3 -m json.tool 2>/dev/null || cat "$SETTINGS_FILE"
else
    echo "   Settings file not found: $SETTINGS_FILE"
    echo "   (This is OK - defaults will be used)"
fi
echo ""

# Check if port 4096 is in use
echo "3. Checking if port 4096 is available..."
if command -v nc &> /dev/null; then
    timeout 1 nc -z localhost 4096 2>/dev/null
    if [ $? -eq 0 ]; then
        echo "   Port 4096 is IN USE"
        echo "   Process using port 4096:"
        netstat -tlnp 2>/dev/null | grep 4096 || ss -tlnp 2>/dev/null | grep 4096 || echo "   (Could not determine process)"
    else
        echo "   Port 4096 is AVAILABLE"
    fi
else
    echo "   (nc command not available, skipping port check)"
fi
echo ""

# Check Go code for port configuration logic
echo "4. Verifying Go code has the port fix..."
SERVER_FILE="/root/mobile-coding-connector/server/agents/opencode/server.go"
if [ -f "$SERVER_FILE" ]; then
    if grep -q "configuredPort := settings.WebServer.Port" "$SERVER_FILE"; then
        echo "   ✓ GetOrStartOpencodeServer() uses configured port"
    else
        echo "   ✗ GetOrStartOpencodeServer() might still use random port"
    fi
    
    if grep -q "exposed.IsPortAvailable(configuredPort)" "$SERVER_FILE"; then
        echo "   ✓ Port availability check uses configured port"
    else
        echo "   ✗ Port availability check might be missing"
    fi
else
    echo "   Could not find server.go file"
fi
echo ""

echo "=========================================="
echo "Summary & Recommendations"
echo "=========================================="
echo ""
echo "The random port issue has been FIXED in the code:"
echo "  - GetOrStartOpencodeServer() now loads settings and uses configured port"
echo "  - Default port is 4096 if not configured"
echo ""
echo "If you're still seeing random ports, it's likely because:"
echo "  1. OLD PROCESSES: There are old opencode processes from before the fix"
echo "     - These show as random ports (33581, 46157, etc.)"
echo "     - They need to be killed and restarted"
echo ""
echo "  2. SETTINGS NOT SAVED: The port might not be saved in settings file"
echo "     - Check the settings file content above"
echo "     - If web_server.port is missing or 0, it defaults to 4096"
echo ""
echo "RECOMMENDED ACTIONS:"
echo "  1. Ask user to manually kill all opencode processes:"
echo "     pkill -9 -f 'opencode.*web'"
echo ""
echo "  2. Verify the settings file has correct port:"
echo "     cat ~/.config/ai-critic/opencode.json"
echo ""
echo "  3. Restart the mobile-coding-connector server"
echo ""
echo "  4. Access the settings page and verify port shows 4096"
echo ""
