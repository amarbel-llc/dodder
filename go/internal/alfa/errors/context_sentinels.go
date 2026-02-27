package errors

import "fmt"

type errContextRetryDisamb struct{}

var errContextRetry = NewWithType[errContextRetryDisamb]("context retry")

type errContextRetryAbortedDisamb struct{}

type errContextRetryAborted struct {
	underlying error
}

func (err errContextRetryAborted) Error() string {
	if err.underlying == nil {
		return "aborted"
	} else {
		return fmt.Sprintf("aborted, %s", err.underlying.Error())
	}
}

func (err errContextRetryAborted) Is(target error) bool {
	_, ok := target.(errContextRetryAborted)
	return ok
}

func (err errContextRetryAborted) GetErrorType() errContextRetryAbortedDisamb {
	return errContextRetryAbortedDisamb{}
}
