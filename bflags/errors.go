package bflags

// ErrorSilencer may be implemented by an input type bound to a command in
// BindRunE for requesting to silence errors.
type ErrorSilencer interface {
	// SilenceErrors returns true to set cobra.Command.SilenceErrors to true
	SilenceErrors() bool
}

// ErrorTraceRemover may be implemented by an input type bound to a command in
// BindRunE for requesting that errors returned by the RunE function have their
// stack trace removed.
type ErrorTraceRemover interface {
	// NoTrace returns true to tell BindRunE to remove error stack trace. Note
	// that this applies only to the execution of the RunE function and not to
	// the setup of the command.
	NoTrace() bool
}
