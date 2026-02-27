package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/box_chars"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/stack_frame"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
)

type encoder struct{}

var CLIErrorTreeEncoder interfaces.EncoderToWriter[error] = encoder{}

func (encoder encoder) EncodeTo(
	input error,
	writer io.Writer,
) (bytesWritten int64, err error) {
	bufferedWriter, repool := pool.GetBufferedWriter(writer)
	defer repool()

	state := &cliTreeState{bufferedWriter: bufferedWriter}
	err = state.encode(input)
	bytesWritten = int64(state.bytesWritten)

	return bytesWritten, err
}

func (encoder encoder) prefixForDepthChild(
	err error,
	depth int,
	lastChild bool,
	continuation bool,
) string {
	if depth == 0 {
		return ""
		// return fmt.Sprintf("%T ", err)
	}

	if continuation {
		return fmt.Sprintf(
			"%s%s ",
			box_chars.PipeVertical,
			strings.Repeat(" ", depth*2),
		)

		// return fmt.Sprintf(
		// 	"%s%s %T ",
		// 	pipeTopBottom,
		// 	strings.Repeat(" ", depth*2),
		// 	err,
		// )
	}

	var firstChar string

	if !lastChild {
		firstChar = box_chars.TeeRight
	} else {
		firstChar = box_chars.ElbowBottomRight
	}

	return fmt.Sprintf(
		"%s%s ",
		firstChar,
		strings.Repeat(box_chars.PipeHorizontal, depth*2),
	)
	// return fmt.Sprintf(
	// 	"%s%s %T ",
	// 	firstChar,
	// 	strings.Repeat(pipeLeftRight, depth*2),
	// 	err,
	// )
}

func (encoder encoder) writeOneErrorMessage(
	err error,
	message string,
	bufferedWriter *bufio.Writer,
	depth int,
	lastChild bool,
) {
	prefixWithoutContinuation := encoder.prefixForDepthChild(
		err,
		depth,
		lastChild,
		false,
	)

	prefixWithContinuation := encoder.prefixForDepthChild(
		err,
		depth,
		lastChild,
		true,
	)

	messageReader := bytes.NewBufferString(message)

	var isEOF bool
	var lineIndex int

	for !isEOF {
		line, err := messageReader.ReadBytes('\n')

		line = bytes.TrimSuffix(line, []byte{'\n'})

		isEOF = err == io.EOF

		if len(line) > 0 {
			if lineIndex == 0 {
				fmt.Fprintf(
					bufferedWriter,
					"%s%s\n",
					prefixWithoutContinuation,
					line,
				)
			} else {
				fmt.Fprintf(
					bufferedWriter,
					"%s%s\n",
					prefixWithContinuation,
					line,
				)
			}
		}

		lineIndex++
	}
}

func (encoder encoder) writeOneChildErrorAndFrame(
	err stack_frame.ErrorAndFrame,
	bufferedWriter *bufio.Writer,
	depth int,
	lastChild bool,
) {
	if err.Err != nil {
		encoder.writeOneErrorMessage(
			err,
			fmt.Sprintf("%s\n%s", err.Err, err.Frame),
			bufferedWriter,
			depth+1,
			lastChild,
		)
	} else {
		encoder.writeOneErrorMessage(
			err,
			err.Frame.String(),
			bufferedWriter,
			depth+1,
			lastChild,
		)
	}
}

func (encoder encoder) encodeToBufferedWriter(
	input error,
	bufferedWriter *bufio.Writer,
	depth int,
	lastChild bool,
) (err error) {
	switch input := input.(type) {
	case stack_frame.ErrorsAndFramesGetter:
		if stackTracer, ok := input.(stack_frame.ErrorStackTracer); ok && !stackTracer.ShouldShowStackTrace() {
			break
		}

		children := input.GetErrorsAndFrames()

		lastChildIdx := len(children) - 1

		for idx, child := range slices.Backward(children) {
			encoder.writeOneChildErrorAndFrame(
				child,
				bufferedWriter,
				depth+1,
				idx == lastChildIdx,
			)
		}

	case errors.UnwrapOne:
		encoder.encodeToBufferedWriter(
			input.Unwrap(),
			bufferedWriter,
			depth+1,
			true,
		)

	case errors.UnwrapMany:
		fmt.Fprintf(bufferedWriter, "%T\n", input)

		children := input.Unwrap()
		lastChildIdx := len(children) - 1

		for idx, child := range slices.Backward(children) {
			encoder.encodeToBufferedWriter(
				child,
				bufferedWriter,
				depth+1,
				idx == lastChildIdx,
			)
		}

	case nil:
		// TODO debug this case
		encoder.writeOneErrorMessage(
			input,
			"error was nil!",
			bufferedWriter,
			depth,
			lastChild,
		)

		return err

	default:
	}

	encoder.writeOneErrorMessage(
		input,
		input.Error(),
		bufferedWriter,
		depth,
		lastChild,
	)

	return err
}
