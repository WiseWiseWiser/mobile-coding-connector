package agentcli

// parseTransferFlags extracts --dry-run from argv and returns the remaining positional args.
func parseTransferFlags(args []string) (dryRun bool, rest []string) {
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
			continue
		}
		rest = append(rest, arg)
	}
	return dryRun, rest
}