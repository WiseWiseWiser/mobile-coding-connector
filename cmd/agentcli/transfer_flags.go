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

// parseUploadFlags extracts upload-specific flags.
func parseUploadFlags(args []string) (dryRun, noCompress, noOverride bool, rest []string) {
	for _, arg := range args {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--no-compress":
			noCompress = true
		case "--no-override":
			noOverride = true
		default:
			rest = append(rest, arg)
		}
	}
	return dryRun, noCompress, noOverride, rest
}