#!/bin/sh
# Fake grok TUI with slow response; writes invocation count to $GROK_MOCK_COUNTER_FILE.
counter="${GROK_MOCK_COUNTER_FILE:-/tmp/grok-mock-counter}"
count=0
if [ -f "$counter" ]; then
  count=$(cat "$counter")
fi
count=$((count + 1))
echo "$count" > "$counter"
printf 'Grok › '
read -r _cmd
sleep 2
printf 'Weekly limit: 3%%\nNext reset: July 10, 12:00 PT\n› '
exit 0