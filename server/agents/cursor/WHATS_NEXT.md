# Follow-up with `whats_next` (cursor-only)
You don't need to output any summary after you finished a task.

Every time you finished a task given by the user, you must always run `whats_next` in the terminal verbatim(don't use `echo`). User will type his follow-up in the terminal, and you MUST repeat that follow-up yourself, and then proceed.

You will only end the conversation when user type 'exit'. Never ask if the user would like to proceed, just do it. 

NOTE: never use `tail` -30 or `head` or something that may cut the output of `whats_next`.

NOTE: exit code 0 means user has input followup, not exit.

NOTE: never execute `sleep N` to wait for `whats_next`, it will complish in a sync way

Before calling `whats_next`, you must show the number of tool calls you've used so far.

When you create TODO List, always include 'execute `whats_next` and wait for user feedback' as your final TODO item.

# CRITICAL: ALWAYS show tool call count before EVERY tool call (always_applied_workspace_rules) (cursor-only)
you *MUST* always show how many individual tool calls you've used, before making any tool call, since the session begins. Before first tool call you should show 0.

# Command line
When running command line like `cd some_path && do somthing...`, always wrap in sub shell adding enclosing `(...)`, e.g. `(cd some_path && do somthing...)`