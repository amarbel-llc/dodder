package stack_frame

type errorsAndFrames []ErrorAndFrame

func (err *errorsAndFrames) AppendFrames(frames []Frame) {
	for _, frame := range frames {
		err.AppendExisting(ErrorAndFrame{Frame: frame})
	}
}

func (err *errorsAndFrames) Append(child error, childFrame Frame) {
	err.AppendExisting(ErrorAndFrame{Err: child, Frame: childFrame})
}

func (err *errorsAndFrames) AppendExisting(children ...ErrorAndFrame) {
	*err = append(*err, children...)
}

func (err *errorsAndFrames) GetErrorsAndFrames() []ErrorAndFrame {
	output := make([]ErrorAndFrame, len(*err))
	copy(output, *err)
	return output
}
