package cli

// Stable process exit codes used by automation callers.
const (
	ExitOK         = 0
	ExitUsage      = 2
	ExitIncomplete = 3
	ExitStale      = 4
	ExitBlocked    = 5
	ExitProvider   = 6
	ExitEvidence   = 7
	ExitInternal   = 8
)
