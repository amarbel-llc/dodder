package errors

func WithoutStack(err error) error {
	if err == nil {
		panic("wrapping empty error")
	}

	return errWithoutStack{underlying: err}
}

type errWithoutStack struct {
	underlying error
}

func (err errWithoutStack) Error() string {
	return err.underlying.Error()
}

func (err errWithoutStack) Unwrap() error {
	return err.underlying
}

func (err errWithoutStack) ShouldShowStackTrace() bool {
	return false
}

func (err errWithoutStack) ShouldHideUnwrap() bool {
	return true
}
