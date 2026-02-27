package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/box_chars"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/stack_frame"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type cliTreeState struct {
	bufferedWriter *bufio.Writer
	bytesWritten   int

	hideStack bool

	stack cliTreeStateStack
}

func (state *cliTreeState) encode(
	input error,
) (err error) {
	var stackTracer stack_frame.ErrorStackTracer

	if errors.As(input, &stackTracer) {
		state.hideStack = !stackTracer.ShouldShowStackTrace()
	}

	state.stack.push(nil, input)
	state.encodeStack()

	if err = state.bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// TODO write instead of return string
func (state *cliTreeState) prefixWithPipesForDepthChild() string {
	depth := state.stack.getDepth()

	if depth == 0 {
		return ""
		// return fmt.Sprintf("%T ", err)
	}

	var count int

	if depth > 0 {
		count = (depth - 1) * 4
	}

	leftPadding := strings.Repeat(" ", count)

	var pipe string

	if state.stack.getLast().isLastChild() {
		pipe = box_chars.ElbowTopRight
	} else {
		pipe = box_chars.TeeRight
	}

	return fmt.Sprintf(
		"%s%s%s%s ",
		leftPadding,
		pipe,
		box_chars.PipeHorizontal,
		box_chars.PipeHorizontal,
	)
}

func (state *cliTreeState) prefixWithoutPipesForDepthChild() string {
	depth := state.stack.getDepth()

	if depth == 0 {
		return ""
	}

	var count int

	if depth > 0 {
		count = (depth - 1) * 4
	}

	leftPadding := strings.Repeat(" ", count)

	return fmt.Sprintf(
		"%s%s     ",
		leftPadding,
		box_chars.PipeVertical,
	)
}

func (state *cliTreeState) writeStrings(values ...string) {
	for _, value := range values {
		bytesWritten, _ := state.bufferedWriter.WriteString(value)
		state.bytesWritten += bytesWritten
	}
}

func (state *cliTreeState) writeBytes(bytess []byte) {
	bytesWritten, _ := state.bufferedWriter.Write(bytess)
	state.bytesWritten += bytesWritten
}

func (state *cliTreeState) writeOneErrorMessage(
	err error,
	message string,
) {
	// TODO firstPrefix depends on whether more than one line is written
	firstPrefix := state.prefixWithPipesForDepthChild()
	remainderPrefix := state.prefixWithoutPipesForDepthChild()
	messageReader := bytes.NewBufferString(message)

	var isEOF bool
	var lineIndex int

	for !isEOF {
		line, err := messageReader.ReadBytes('\n')

		line = bytes.TrimSuffix(line, []byte{'\n'})

		isEOF = err == io.EOF

		if len(line) > 0 {
			if lineIndex > 0 {
				state.writeStrings(remainderPrefix)
			} else {
				state.writeStrings(firstPrefix)
			}

			state.writeBytes(line)
			state.writeStrings("\n")
		}

		lineIndex++
	}
}

func (state *cliTreeState) writeOneChildErrorAndFrame(
	err stack_frame.ErrorAndFrame,
) {
	if err.Err != nil {
		state.writeOneErrorMessage(
			err,
			fmt.Sprintf("%s\n%s", err.Err, err.Frame),
		)
	} else {
		state.writeOneErrorMessage(err, err.Frame.String())
	}
}

// TODO separate tree transformation from writing
func (state *cliTreeState) encodeStack() {
	stackItem := state.stack.getLast()
	input := stackItem.child

	switch inputTyped := input.(type) {
	case interfaces.ErrorHiddenWrapper:
		if inputTyped.ShouldHideUnwrap() {
			child := inputTyped.Unwrap()

			if child != nil {
				stackItem.child = child
				state.encodeStack()
			}
		} else {
			state.printErrorOneUnwrapper(inputTyped)
		}

	case stack_frame.ErrorsAndFramesGetter:
		if state.hideStack {
			child := errors.Unwrap(input)
			stackItem.child = child
			state.encodeStack()
			return
		}

		{
			root := inputTyped.GetErrorRoot()
			stackItem.child = root
			state.encodeStack()
		}

		stackItem.child = input

		children := inputTyped.GetErrorsAndFrames()

		childStackItem := state.stack.push(input, nil)
		childStackItem.childCount = len(children)

		var child stack_frame.ErrorAndFrame

		for childStackItem.childIdx, child = range children {
			childStackItem.child = child
			state.writeOneChildErrorAndFrame(child)
		}

	case errors.UnwrapOne:
		state.printErrorOneUnwrapper(inputTyped)

	case errors.UnwrapMany:
		children := inputTyped.Unwrap()

		if len(children) == 1 {
			stackItem.child = children[0]
			state.encodeStack()
			return
		}

		state.writeOneErrorMessage(
			input,
			// fmt.Sprintf("%T: %s", input, input.Error()),
			fmt.Sprintf("%s", input.Error()),
		)

		childStackItem := state.stack.push(input, nil)
		childStackItem.childCount = len(children)

		for childStackItem.childIdx, childStackItem.child = range children {
			state.encodeStack()
		}

	case nil:
		state.writeOneErrorMessage(
			inputTyped,
			"error was nil!",
		)

		return

	default:
		state.writeOneErrorMessage(
			input,
			input.Error(),
		)
	}
}

func (state cliTreeState) printErrorOneUnwrapper(err errors.UnwrapOne) {
	state.printErrorOneUnwrapperWithChild(err, err.Unwrap())
}

func (state cliTreeState) printErrorOneUnwrapperWithChild(
	err error,
	child error,
) {
	if child == nil {
		state.writeOneErrorMessage(err, err.Error())
		return
	}

	state.writeOneErrorMessage(
		err,
		// fmt.Sprintf("%T: %s", err, err.Error()),
		fmt.Sprintf("%s", err.Error()),
	)

	childStackItem := state.stack.push(err, child)
	childStackItem.childCount = 1
	state.encodeStack()
	state.stack.pop()
}
