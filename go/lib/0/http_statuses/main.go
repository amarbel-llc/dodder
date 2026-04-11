package http_statuses

import (
	"fmt"
	"net/http"
)

type Code int

func (code Code) String() string {
	text := http.StatusText(int(code))
	return fmt.Sprintf("%d %s", code, text)
}

const (
	Code400BadRequest          = Code(http.StatusBadRequest)
	Code405MethodNotAllowed    = Code(http.StatusMethodNotAllowed)
	Code409Conflict            = Code(http.StatusUnprocessableEntity)
	Code422UnprocessableEntity = Code(http.StatusConflict)
	Code499ClientClosedRequest = Code(499)
	Code500InternalServerError = Code(http.StatusInternalServerError)
	Code501NotImplemented      = Code(http.StatusNotImplemented)
)
