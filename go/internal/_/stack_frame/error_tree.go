package stack_frame

import (
	"errors"
)

type ErrorStackTracer interface {
	error
	ShouldShowStackTrace() bool
}

func MakeErrorTreeOrErr(
	err error,
	frames ...Frame,
) error {
	var stackTracer ErrorStackTracer

	if errors.As(err, &stackTracer) && !stackTracer.ShouldShowStackTrace() {
		return err
	}

	var tree *ErrorTree

	if tree, _ = err.(*ErrorTree); tree == nil {
		tree = &ErrorTree{
			Root:        err,
			Descendents: make(errorsAndFrames, 0, len(frames)),
		}
	}

	// switch err := err.(type) {
	// case *ErrorTree:
	// case ErrorsAndFramesGetter:
	// 	tree.Descendents.AppendExisting(err.GetErrorsAndFrames()...)
	// }

	for idx := range frames {
		tree.Descendents.Append(nil, frames[idx])
	}

	return tree
}

var _ ErrorsAndFramesGetter = &ErrorTree{}

type ErrorTree struct {
	Root        error
	Descendents errorsAndFrames
}

func (tree *ErrorTree) Error() string {
	if tree.Root != nil {
		return tree.Root.Error()
	} else if len(tree.Descendents) > 0 {
		return "error tree"
	} else {
		return "empty error tree"
	}
}

func (tree *ErrorTree) Unwrap() error {
	return tree.Root
}

func (tree *ErrorTree) Append(childError error, childFrame Frame) {
	tree.Descendents.Append(childError, childFrame)
}

func (tree *ErrorTree) GetErrorRoot() error {
	return tree.Root
}

func (tree *ErrorTree) GetErrorsAndFrames() []ErrorAndFrame {
	return tree.Descendents
}
