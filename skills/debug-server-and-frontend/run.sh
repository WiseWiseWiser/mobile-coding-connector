#!/bin/bash
# Quick Test Skill Wrapper for Server and Frontend
# This script wraps the Node.js test runner for easy execution

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print banner
echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════╗"
echo "║          Quick Test: Server and Frontend                     ║"
echo "║          Puppeteer-based Browser Testing                 ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo -e "${RED}Error: Node.js is not installed${NC}"
    exit 1
fi

# Check if dependencies are installed
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing dependencies...${NC}"
    npm install
fi

# Parse arguments
TEST_URL="${TEST_URL:-https://agent-fast-apex-nest-23aed.xhd2015.xyz}"
HEADLESS="${HEADLESS:-true}"
DEBUG="${DEBUG:-false}"

# Show usage if --help is passed
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --url URL       Set the test URL (default: $TEST_URL)"
    echo "  --headed        Run with visible browser window"
    echo "  --debug         Enable debug output"
    echo "  --help, -h      Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  TEST_URL        Base URL for testing"
    echo "  HEADLESS        Run headless (true/false)"
    echo "  DEBUG           Enable debug mode (true/false)"
    exit 0
fi

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --url)
            TEST_URL="$2"
            shift 2
            ;;
        --headed)
            HEADLESS="false"
            shift
            ;;
        --debug)
            DEBUG="true"
            shift
            ;;
        *)
            echo -e "${YELLOW}Warning: Unknown option $1${NC}"
            shift
            ;;
    esac
done

echo -e "${BLUE}Configuration:${NC}"
echo "  URL:      $TEST_URL"
echo "  Headless: $HEADLESS"
echo "  Debug:    $DEBUG"
echo ""

# Export environment variables
export TEST_URL
export HEADLESS
export DEBUG

# Run the test
echo -e "${GREEN}Starting tests...${NC}"
echo ""

node test-opencode-settings.js

exit_code=$?

echo ""
if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
else
    echo -e "${RED}✗ Some tests failed${NC}"
fi

exit $exit_code
