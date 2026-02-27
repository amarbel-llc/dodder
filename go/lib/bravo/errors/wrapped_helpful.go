package errors

type (
	Helpful interface {
		error
		GetErrorCause() []string
		GetErrorRecovery() []string
	}
)

func WithHelp(
	err error,
	cause []string,
	recovery []string,
) Helpful {
	return helpful{
		underlying: err,
		cause:      cause,
		recovery:   recovery,
	}
}

type helpful struct {
	underlying error
	cause      []string
	recovery   []string
}

func (err helpful) Error() string {
	return err.underlying.Error()
}

func (err helpful) Unwrap() error {
	return err.underlying
}

func (err helpful) GetErrorCause() []string {
	return err.cause
}

func (err helpful) GetErrorRecovery() []string {
	return err.recovery
}

func (err helpful) ShouldShowStackTrace() bool {
	return false
}

func (err helpful) ShouldHideUnwrap() bool {
	return true
}
