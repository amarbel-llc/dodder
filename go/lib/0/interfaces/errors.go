package interfaces

type (
	ErrorOneUnwrapper interface {
		error
		Unwrap() error
	}

	ErrorManyUnwrapper interface {
		error
		Unwrap() []error
	}

	// When printing error trees, `error_coders` uses the presence of
	// `ShouldHideUnwrap()` and its return value to determine if the parent
	// error should be printed.
	ErrorHiddenWrapper interface {
		ErrorOneUnwrapper
		ShouldHideUnwrap() bool
	}
)
