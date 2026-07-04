#!/bin/sh
# Writes invocation count to $GROK_MOCK_COUNTER_FILE for overlap detection.
counter="${GROK_MOCK_COUNTER_FILE:-/tmp/grok-mock-counter}"
count=0
if [ -f "$counter" ]; then
  count=$(cat "$counter")
fi
count=$((count + 1))
echo "$count" > "$counter"
sleep 2
echo "Weekly limit: 3%"
echo "Next reset: July 10, 12:00 PT"
exit 0