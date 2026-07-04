#!/bin/sh
# Fake grok TUI that fails after prompt (stderr + exit 1).
printf 'Grok › '
read -r _cmd
echo "error: grok usage unavailable" >&2
exit 1