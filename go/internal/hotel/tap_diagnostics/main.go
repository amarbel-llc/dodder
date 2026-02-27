package tap_diagnostics

import (
	"errors"
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

func FromError(err error) map[string]string {
	diag := map[string]string{
		"severity": "fail",
		"message":  err.Error(),
	}

	var errNotEqual markl.ErrNotEqual
	if errors.As(err, &errNotEqual) {
		diag["expected"] = fmt.Sprintf("%s", errNotEqual.Expected)
		diag["actual"] = fmt.Sprintf("%s", errNotEqual.Actual)
		return diag
	}

	var errIsNull markl.ErrIsNull
	if errors.As(err, &errIsNull) {
		diag["field"] = errIsNull.Purpose
		return diag
	}

	return diag
}
