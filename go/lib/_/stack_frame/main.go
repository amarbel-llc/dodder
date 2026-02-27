package stack_frame

//go:noinline
func Wrap(err error) error {
	frame, _ := MakeFrame(1)
	return frame.Wrap(err)
}

//go:noinline
func WrapSkip(skip int, err error) error {
	frame, _ := MakeFrame(skip + 1)
	return frame.Wrap(err)
}

//go:noinline
func Wrapf(err error, format string, args ...any) error {
	frame, _ := MakeFrame(1)
	return frame.Wrapf(err, format, args...)
}

//go:noinline
func Errorf(format string, args ...any) error {
	frame, _ := MakeFrame(1)
	return frame.Errorf(format, args...)
}
