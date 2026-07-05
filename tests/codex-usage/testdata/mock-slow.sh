#!/bin/sh
# Writes invocation count to $CODEX_MOCK_COUNTER_FILE for overlap detection.
counter="${CODEX_MOCK_COUNTER_FILE:-/tmp/codex-mock-counter}"
count=0
if [ -f "$counter" ]; then
  count=$(cat "$counter")
fi
count=$((count + 1))
echo "$count" > "$counter"
sleep 2
echo "Monthly usage: 30%"
echo "Credits used: 1000 of 5000"
echo "Next reset: 10:00 on 2 Aug"
exit 0