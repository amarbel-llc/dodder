package stack_frame

type (
	ErrorAndFrame struct {
		Err   error
		Frame Frame
	}

	ErrorsAndFramesGetter interface {
		GetErrorRoot() error
		GetErrorsAndFrames() []ErrorAndFrame
	}
)

func (err ErrorAndFrame) IsEmpty() bool {
	return err.Err == nil && !err.Frame.nonZero
}

func (err ErrorAndFrame) Error() string {
	if err.Err == nil {
		return "stack frame with no error"
	} else {
		return err.Err.Error()
	}
}

func (err ErrorAndFrame) Unwrap() error {
	return err.Err
}
