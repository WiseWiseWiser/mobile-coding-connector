#!/bin/sh
# Fake grok TUI for GROK_SHOW_USAGE_COMMAND: prompt, read /usage show, print usage lines.
printf 'Grok › '
read -r _cmd
printf 'Weekly limit: 6%%\nNext reset: July 9, 16:55 PT\n› '
exit 0